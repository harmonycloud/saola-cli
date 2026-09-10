// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package ssamigrate

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

const ApplyManager = "opensaola"
const ToolManager = "saola-ssa-migration"

// ReceiptAnnotation and ConvergedAnnotation must stay byte-identical to
// AnnotationSSACliMigration/AnnotationSSACliConverged in opensaola internal/k8s.
const ReceiptAnnotation = "middleware.cn/ssa-cli-migration"
const ConvergedAnnotation = "middleware.cn/ssa-cli-converged"

func digest(v any) string { b, _ := json.Marshal(v); return fmt.Sprintf("sha256:%x", sha256.Sum256(b)) }

// selectApplyEntry picks the OpenSaola Apply entry for converged-hash comparison.
// Multiple entries can coexist briefly across API version migrations; prefer the
// object's current API version, then the lowest version, so both this CLI and the
// OpenSaola controller hash the same entry. Keep in sync with the controller.
func selectApplyEntry(live *unstructured.Unstructured) (metav1.ManagedFieldsEntry, bool) {
	var entries []metav1.ManagedFieldsEntry
	for _, f := range live.GetManagedFields() {
		if f.Manager == ApplyManager && f.Operation == metav1.ManagedFieldsOperationApply && f.Subresource == "" {
			entries = append(entries, f)
		}
	}
	if len(entries) == 0 {
		return metav1.ManagedFieldsEntry{}, false
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].APIVersion < entries[j].APIVersion })
	for _, f := range entries {
		if f.APIVersion == live.GetAPIVersion() {
			return f, true
		}
	}
	return entries[0], true
}
func fieldSet(e metav1.ManagedFieldsEntry) (*fieldpath.Set, error) {
	if e.FieldsType != "FieldsV1" || e.FieldsV1 == nil {
		return nil, fmt.Errorf("%s has unsupported fields", e.Manager)
	}
	s := &fieldpath.Set{}
	if err := s.FromJSON(bytes.NewReader(e.FieldsV1.Raw)); err != nil {
		return nil, err
	}
	return s, nil
}
func rawSet(s *fieldpath.Set) json.RawMessage { b, _ := s.ToJSON(); return b }
func paths(s *fieldpath.Set) []string {
	out := []string{}
	s.Iterate(func(p fieldpath.Path) { out = append(out, p.String()) })
	sort.Strings(out)
	return out
}

// Schemas retains historical merge topology instead of inferring it from current templates.
// Schemas 保留历史合并拓扑，不从当前模板反推历史字段。
type Schemas struct {
	Definitions map[string]*spec.Schema `json:"definitions"`
	converter   managedfields.TypeConverter
}

func ReadSchemas(raw []byte) (*Schemas, error) {
	s := &Schemas{}
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, err
	}
	if len(s.Definitions) == 0 {
		return nil, fmt.Errorf("OpenAPI v2 definitions missing")
	}
	tc, err := managedfields.NewTypeConverter(s.Definitions, false)
	if err != nil {
		return nil, err
	}
	s.converter = tc
	return s, nil
}
func (s *Schemas) Hash(apiVersion, kind string) (string, error) {
	root := ""
	for name, def := range s.Definitions {
		ext := def.Extensions["x-kubernetes-group-version-kind"]
		raw, _ := json.Marshal(ext)
		var gvks []struct{ Group, Version, Kind string }
		_ = json.Unmarshal(raw, &gvks)
		for _, g := range gvks {
			av := g.Version
			if g.Group != "" {
				av = g.Group + "/" + g.Version
			}
			if av == apiVersion && g.Kind == kind {
				if root != "" {
					return "", fmt.Errorf("ambiguous schema")
				}
				root = name
			}
		}
	}
	if root == "" {
		return "", fmt.Errorf("schema missing for %s %s", apiVersion, kind)
	}
	closure := map[string]*spec.Schema{}
	var visit func(string) error
	visit = func(name string) error {
		if _, ok := closure[name]; ok {
			return nil
		}
		d, ok := s.Definitions[name]
		if !ok {
			return fmt.Errorf("missing schema reference %s", name)
		}
		closure[name] = d
		raw, _ := json.Marshal(d)
		var v any
		_ = json.Unmarshal(raw, &v)
		var walk func(any) error
		walk = func(v any) error {
			switch x := v.(type) {
			case map[string]any:
				for k, y := range x {
					if k == "$ref" {
						r, ok := y.(string)
						if !ok || !strings.HasPrefix(r, "#/definitions/") {
							return fmt.Errorf("unsupported schema ref")
						}
						if err := visit(strings.TrimPrefix(r, "#/definitions/")); err != nil {
							return err
						}
					} else if err := walk(y); err != nil {
						return err
					}
				}
			case []any:
				for _, y := range x {
					if err := walk(y); err != nil {
						return err
					}
				}
			}
			return nil
		}
		return walk(v)
	}
	if err := visit(root); err != nil {
		return "", err
	}
	return digest(closure), nil
}
func intent(o *unstructured.Unstructured) *unstructured.Unstructured {
	c := o.DeepCopy()
	delete(c.Object, "status")
	meta := map[string]any{}
	if m, ok := c.Object["metadata"].(map[string]any); ok {
		for _, k := range []string{"labels", "annotations"} {
			if v, ok := m[k]; ok {
				meta[k] = v
			}
		}
	}
	c.Object["metadata"] = meta
	a := c.GetAnnotations()
	for _, k := range []string{ReceiptAnnotation, ConvergedAnnotation, zeusv1.LabelConfigurations, "middleware.cn/configuration-owner-uid", "middleware.cn/configuration-uid", "middleware.cn/ssa-compatible-generation"} {
		delete(a, k)
	}
	if len(a) == 0 {
		unstructured.RemoveNestedField(c.Object, "metadata", "annotations")
	} else {
		c.SetAnnotations(a)
	}
	return c
}

// ProvenFields excludes admission changes and values changed since the audited create.
// ProvenFields 排除准入改写及创建后值已改变的字段，避免同名 manager 混入的写入被接管。
func (s *Schemas) ProvenFields(request, response, live *unstructured.Unstructured) (*fieldpath.Set, error) {
	req, err := s.converter.ObjectToTyped(intent(request))
	if err != nil {
		return nil, err
	}
	res, err := s.converter.ObjectToTyped(intent(response))
	if err != nil {
		return nil, err
	}
	now, err := s.converter.ObjectToTyped(intent(live))
	if err != nil {
		return nil, err
	}
	fields, err := req.ToFieldSet()
	if err != nil {
		return nil, err
	}
	changed, err := req.Compare(res)
	if err != nil {
		return nil, err
	}
	fields = fields.RecursiveDifference(changed.Modified.Union(changed.Removed))
	changed, err = req.Compare(now)
	if err != nil {
		return nil, err
	}
	return fields.RecursiveDifference(changed.Modified.Union(changed.Removed)), nil
}

type Transfer struct {
	Manager    string          `json:"manager"`
	Fields     json.RawMessage `json:"fieldsV1"`
	BeforeHash string          `json:"beforeHash"`
}

// TransferFields splits Update ownership; every unselected workflow stays intact.
// TransferFields 只拆分选中的 Update 字段，其他工作流保持原样。
func TransferFields(live *unstructured.Unstructured, transfers []Transfer) ([]metav1.ManagedFieldsEntry, error) {
	entries := live.GetManagedFields()
	result := []metav1.ManagedFieldsEntry{}
	union := &fieldpath.Set{}
	seen := map[string]bool{}
	applyFound := false
	target := metav1.ManagedFieldsEntry{Manager: ApplyManager, Operation: metav1.ManagedFieldsOperationApply, APIVersion: live.GetAPIVersion(), FieldsType: "FieldsV1"}
	for _, e := range entries {
		if e.Subresource != "" {
			result = append(result, e)
			continue
		}
		if e.Manager == ApplyManager && e.Operation == metav1.ManagedFieldsOperationApply {
			if applyFound || e.APIVersion != live.GetAPIVersion() {
				return nil, fmt.Errorf("ambiguous/cross-version Apply workflow")
			}
			f, err := fieldSet(e)
			if err != nil {
				return nil, err
			}
			union = union.Union(f)
			target = *e.DeepCopy()
			applyFound = true
			continue
		}
		chosen := false
		for _, t := range transfers {
			if e.Manager != t.Manager || e.Operation != metav1.ManagedFieldsOperationUpdate {
				continue
			}
			if seen[t.Manager] || e.APIVersion != live.GetAPIVersion() {
				return nil, fmt.Errorf("ambiguous/cross-version Update workflow")
			}
			seen[t.Manager] = true
			f, err := fieldSet(e)
			if err != nil {
				return nil, err
			}
			if digest(rawSet(f)) != t.BeforeHash {
				return nil, fmt.Errorf("legacy fieldset changed; regenerate plan")
			}
			take := &fieldpath.Set{}
			if err = take.FromJSON(bytes.NewReader(t.Fields)); err != nil {
				return nil, err
			}
			if take.Empty() || !take.Difference(f).Empty() {
				return nil, fmt.Errorf("transfer is not a nonempty subset")
			}
			union = union.Union(take)
			rest := f.Difference(take)
			if !rest.Empty() {
				e.FieldsV1 = &metav1.FieldsV1{Raw: rawSet(rest)}
				result = append(result, e)
			}
			chosen = true
			break
		}
		if !chosen {
			result = append(result, e)
		}
	}
	if len(seen) != len(transfers) {
		return nil, fmt.Errorf("legacy workflow missing")
	}
	target.FieldsV1 = &metav1.FieldsV1{Raw: rawSet(union)}
	result = append(result, target)
	return result, nil
}
