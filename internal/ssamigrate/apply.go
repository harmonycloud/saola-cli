// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package ssamigrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

type Result struct {
	Resource Ref    `json:"resource"`
	State    string `json:"state"`
	Error    string `json:"error,omitempty"`
}

func receipt(p *Plan, item Entry) string {
	return digest(struct {
		Plan string
		UID  types.UID
	}{p.ID, item.Resource.UID})
}

// SaveExclusive protects plans and backups containing potentially sensitive metadata.
// SaveExclusive 以独占、0600 权限保存计划和备份，不覆盖原始文件。
func SaveExclusive(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path) // never leave a truncated file behind to poison retries
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}
func (e *Engine) checkPlan(ctx context.Context, p *Plan) error {
	if !p.Valid() {
		return fmt.Errorf("invalid or edited plan; regenerate it")
	}
	uid, err := e.ClusterUID(ctx)
	if err != nil {
		return err
	}
	if uid == "" || uid != p.ClusterUID {
		return fmt.Errorf("cluster UID mismatch")
	}
	return nil
}
func (e *Engine) readOwner(ctx context.Context, item Entry, live *unstructured.Unstructured) error {
	o := item.Owner.Object()
	if err := e.Client.Get(ctx, client.ObjectKeyFromObject(o), o); err != nil {
		return err
	}
	if o.GetUID() != item.Owner.UID {
		return fmt.Errorf("owner replaced")
	}
	if err := verifyOwner(live, o); err != nil {
		return err
	}
	if live.GetAnnotations()[zeusv1.LabelConfigurations] != item.Configuration {
		return fmt.Errorf("Configuration identity changed")
	}
	return e.verifySource(ctx, o, live)
}
func (e *Engine) read(ctx context.Context, item Entry) (*unstructured.Unstructured, error) {
	o := item.Resource.Object()
	if err := e.Client.Get(ctx, client.ObjectKeyFromObject(o), o); err != nil {
		return nil, err
	}
	if o.GetUID() != item.Resource.UID {
		return nil, fmt.Errorf("resource replaced")
	}
	return o, e.readOwner(ctx, item, o)
}

// Apply changes metadata ownership only; OpenSaola performs subsequent desired-state Apply.
// Apply 只迁移 metadata 字段归属，后续期望态由 OpenSaola 正常 Apply。
func (e *Engine) Apply(ctx context.Context, p *Plan, backupDir string) ([]Result, error) {
	if err := e.checkPlan(ctx, p); err != nil {
		return nil, err
	}
	if len(p.Errors) > 0 {
		return nil, fmt.Errorf("scan was incomplete; resolve discovery/list errors and regenerate plan")
	}
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Stat(backupDir)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("backup directory must not be accessible to group/others")
	}
	results := []Result{}
	var failures []error
	for _, item := range p.Entries {
		r := Result{Resource: item.Resource, State: item.Status}
		if item.Status != "ready" {
			if item.Status == "blocked" {
				r.Error = item.Reason
				failures = append(failures, fmt.Errorf("%s: %s", item.Resource.Name, item.Reason))
			}
			results = append(results, r)
			continue
		}
		already, err := e.applyOne(ctx, p, item, backupDir)
		if err != nil {
			r.State = "failed"
			r.Error = err.Error()
			failures = append(failures, err)
		} else if already {
			r.State = "already-migrated"
		} else {
			r.State = "migrated"
		}
		results = append(results, r)
		if ctx.Err() != nil {
			failures = append(failures, fmt.Errorf("migration interrupted: %w", ctx.Err()))
			break
		}
	}
	return results, errors.Join(failures...)
}
func (e *Engine) applyOne(ctx context.Context, p *Plan, item Entry, backupDir string) (bool, error) {
	hash, err := e.Current.Hash(item.Resource.APIVersion, item.Resource.Kind)
	if err != nil {
		return false, err
	}
	if hash != item.SchemaHash {
		return false, fmt.Errorf("schema changed since plan")
	}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 && e.RefreshSchemas != nil {
			current, err := e.RefreshSchemas(ctx)
			if err != nil {
				return false, err
			}
			hash, err := current.Hash(item.Resource.APIVersion, item.Resource.Kind)
			if err != nil {
				return false, err
			}
			if hash != item.SchemaHash {
				return false, fmt.Errorf("schema changed during migration")
			}
			e.Current = current
		}
		live, err := e.read(ctx, item)
		if err != nil {
			return false, err
		}
		if live.GetAnnotations()[ReceiptAnnotation] == receipt(p, item) {
			return true, e.verifyTransferred(live, item)
		}
		if live.GetAnnotations()[ReceiptAnnotation] != "" {
			return false, fmt.Errorf("another migration receipt exists")
		}
		if businessHash(live) != item.ObjectHash || fieldsHash(live.GetManagedFields()) != fieldsHash(item.BeforeFields) {
			return false, fmt.Errorf("object or ownership changed since plan; regenerate plan")
		}
		after, err := TransferFields(live, item.Transfers)
		if err != nil {
			return false, err
		}
		if fieldsHash(after) != fieldsHash(item.AfterFields) {
			return false, fmt.Errorf("plan result does not match transfer")
		}
		annotations := live.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[ReceiptAnnotation] = receipt(p, item)
		patch, _ := json.Marshal([]map[string]any{{"op": "test", "path": "/metadata/uid", "value": live.GetUID()}, {"op": "test", "path": "/metadata/resourceVersion", "value": live.GetResourceVersion()}, {"op": "replace", "path": "/metadata/managedFields", "value": after}, {"op": "add", "path": "/metadata/annotations", "value": annotations}})
		preview := live.DeepCopy()
		if err = e.Client.Patch(ctx, preview, client.RawPatch(types.JSONPatchType, patch), client.FieldOwner(ToolManager), client.DryRunAll); err != nil {
			if apierrors.IsConflict(err) {
				lastErr = err
				continue
			}
			// Invalid is deterministic (validation/admission); retrying cannot help.
			return false, err
		}
		if err = verifyMetadataOnly(live, preview, after); err != nil {
			return false, fmt.Errorf("dry-run: %w", err)
		}
		path := filepath.Join(backupDir, string(item.Resource.UID)+".json")
		if err = SaveExclusive(path, live.Object); err != nil {
			if !os.IsExist(err) {
				return false, err
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return false, readErr
			}
			old := &unstructured.Unstructured{}
			if err = json.Unmarshal(raw, &old.Object); err != nil {
				return false, err
			}
			if old.GetUID() != live.GetUID() || businessHash(old) != item.ObjectHash || fieldsHash(old.GetManagedFields()) != fieldsHash(item.BeforeFields) {
				return false, fmt.Errorf("existing backup does not match plan")
			}
		}
		updated := live.DeepCopy()
		if err = e.Client.Patch(ctx, updated, client.RawPatch(types.JSONPatchType, patch), client.FieldOwner(ToolManager)); err != nil {
			if apierrors.IsConflict(err) {
				lastErr = err
				continue
			}
			return false, err
		}
		if err = verifyMetadataOnly(live, updated, after); err != nil {
			return false, fmt.Errorf("migration written but verification failed (backup retained): %w", err)
		}
		observed, err := e.read(ctx, item)
		if err != nil {
			return false, err
		}
		if observed.GetAnnotations()[ReceiptAnnotation] != receipt(p, item) {
			return false, fmt.Errorf("migration receipt missing")
		}
		return false, e.verifyTransferred(observed, item)
	}
	return false, fmt.Errorf("resource changed during five attempts (last error: %w); retry with a fresh plan if necessary", lastErr)
}
func verifyMetadataOnly(before, after *unstructured.Unstructured, want []metav1.ManagedFieldsEntry) error {
	if businessHash(before) != businessHash(after) || digest(before.Object["status"]) != digest(after.Object["status"]) {
		return fmt.Errorf("API server/admission changed business fields or status")
	}
	actual := []metav1.ManagedFieldsEntry{}
	for _, f := range after.GetManagedFields() {
		if f.Manager == ToolManager && f.Operation == metav1.ManagedFieldsOperationUpdate && f.Subresource == "" {
			continue
		}
		actual = append(actual, f)
	}
	if fieldsHash(actual) != fieldsHash(want) {
		return fmt.Errorf("API server changed unexpected managedFields")
	}
	return nil
}
func (e *Engine) verifyTransferred(live *unstructured.Unstructured, item Entry) error {
	var converged struct {
		Migration         string `json:"migration"`
		AppliedFieldsHash string `json:"appliedFieldsHash"`
	}
	_ = json.Unmarshal([]byte(live.GetAnnotations()[ConvergedAnnotation]), &converged)
	currentApplyHash := ""
	if f, ok := selectApplyEntry(live); ok {
		currentApplyHash = digest(struct {
			APIVersion string           `json:"apiVersion"`
			FieldsV1   *metav1.FieldsV1 `json:"fieldsV1"`
		}{f.APIVersion, f.FieldsV1})
	}
	completed := converged.Migration != "" && converged.Migration == live.GetAnnotations()[ReceiptAnnotation] && currentApplyHash != "" && currentApplyHash == converged.AppliedFieldsHash
	presentValue, err := e.Current.converter.ObjectToTyped(intent(live))
	if err != nil {
		return err
	}
	present, err := presentValue.ToFieldSet()
	if err != nil {
		return err
	}
	owned := &fieldpath.Set{}
	for _, f := range live.GetManagedFields() {
		if f.Subresource != "" || f.Manager == ToolManager {
			continue
		}
		selected := false
		for _, t := range item.Transfers {
			if f.Manager == t.Manager && f.Operation == metav1.ManagedFieldsOperationUpdate {
				selected = true
			}
		}
		if selected {
			continue
		}
		if f.APIVersion != live.GetAPIVersion() {
			return fmt.Errorf("cannot verify cross-version ownership")
		}
		set, err := fieldSet(f)
		if err != nil {
			return err
		}
		owned = owned.Union(set)
	}
	transferSets := make([]*fieldpath.Set, len(item.Transfers))
	for i, t := range item.Transfers {
		set, err := fieldSet(metav1.ManagedFieldsEntry{Manager: t.Manager, FieldsType: "FieldsV1", FieldsV1: &metav1.FieldsV1{Raw: t.Fields}})
		if err != nil {
			return err
		}
		transferSets[i] = set
	}
	for i := range item.Transfers {
		if !completed {
			lost := present.Intersection(transferSets[i]).Difference(owned)
			if !lost.Empty() {
				return fmt.Errorf("transferred fields still exist but their ownership disappeared: %v", paths(lost))
			}
		}
	}
	for i, t := range item.Transfers {
		for _, f := range live.GetManagedFields() {
			if f.Subresource != "" || f.Manager != t.Manager || f.Operation != metav1.ManagedFieldsOperationUpdate {
				continue
			}
			s, err := fieldSet(f)
			if err != nil {
				return err
			}
			if !s.Intersection(transferSets[i]).Empty() {
				return fmt.Errorf("transferred fields reappeared in legacy workflow")
			}
		}
	}
	return nil
}
func (e *Engine) Verify(ctx context.Context, p *Plan) ([]Result, error) {
	if err := e.checkPlan(ctx, p); err != nil {
		return nil, err
	}
	out := []Result{}
	var failures []error
	for _, item := range p.Entries {
		r := Result{Resource: item.Resource, State: item.Status}
		if item.Status == "ready" {
			live, err := e.read(ctx, item)
			if err == nil && live.GetAnnotations()[ReceiptAnnotation] != receipt(p, item) {
				err = fmt.Errorf("migration not completed")
			}
			if err == nil {
				err = e.verifyTransferred(live, item)
			}
			if err != nil {
				r.State = "failed"
				r.Error = err.Error()
				failures = append(failures, err)
			} else {
				r.State = "ownership-migrated"
			}
		} else if item.Status == "blocked" {
			r.Error = item.Reason
			failures = append(failures, fmt.Errorf("blocked: %s", item.Resource.Name))
		}
		out = append(out, r)
	}
	return out, errors.Join(failures...)
}
