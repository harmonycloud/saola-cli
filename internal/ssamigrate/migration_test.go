// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package ssamigrate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harmonycloud/saola-cli/internal/packager"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func entry(manager string, operation metav1.ManagedFieldsOperationType, fields string) metav1.ManagedFieldsEntry {
	return metav1.ManagedFieldsEntry{Manager: manager, Operation: operation, APIVersion: "v1", FieldsType: "FieldsV1", FieldsV1: &metav1.FieldsV1{Raw: []byte(fields)}}
}
func TestTransferSplitsLegacyAndPreservesExternal(t *testing.T) {
	old := entry("manager", metav1.ManagedFieldsOperationUpdate, `{"f:data":{"f:legacy":{},"f:manual":{}}}`)
	ext := entry("human", metav1.ManagedFieldsOperationApply, `{"f:data":{"f:shared":{}}}`)
	apply := entry(ApplyManager, metav1.ManagedFieldsOperationApply, `{"f:data":{"f:keep":{}}}`)
	o := &unstructured.Unstructured{}
	o.SetAPIVersion("v1")
	o.SetKind("ConfigMap")
	o.SetManagedFields([]metav1.ManagedFieldsEntry{old, ext, apply})
	s, _ := fieldSet(old)
	result, err := TransferFields(o, []Transfer{{Manager: "manager", BeforeHash: digest(rawSet(s)), Fields: json.RawMessage(`{"f:data":{"f:legacy":{}}}`)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatal(result)
	}
	f, _ := fieldSet(result[0])
	if got := paths(f); len(got) != 1 || got[0] != ".data.manual" {
		t.Fatal(got)
	}
	if fieldsHash(result[1:2]) != fieldsHash([]metav1.ManagedFieldsEntry{ext}) {
		t.Fatal("external workflow changed")
	}
	f, _ = fieldSet(result[2])
	if len(paths(f)) != 2 {
		t.Fatal(paths(f))
	}
	original, _ := fieldSet(o.GetManagedFields()[0])
	if len(paths(original)) != 2 {
		t.Fatal("input mutated")
	}
}
func TestPlanTamperingRejected(t *testing.T) {
	p := &Plan{Version: 1, ClusterUID: "cluster", Writer: "writer"}
	p.Seal()
	if !p.Valid() {
		t.Fatal("invalid fresh plan")
	}
	p.Writer = "other"
	if p.Valid() {
		t.Fatal("edited plan accepted")
	}
}

func TestAuditPatchBodiesDoNotRequireKind(t *testing.T) {
	for _, body := range []string{`{"metadata":{"labels":{"x":"y"}}}`, `[{"op":"remove","path":"/spec/parameters"}]`} {
		var event AuditEvent
		if err := json.Unmarshal([]byte(`{"stage":"ResponseComplete","verb":"patch","requestObject":`+body+`}`), &event); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceFallsBackToOwnerPackage(t *testing.T) {
	dir := t.TempDir()
	for file, content := range map[string]string{"metadata.yaml": "name: old\nversion: 1.0.0\n", "baseline.yaml": "apiVersion: middleware.cn/v1\nkind: MiddlewareBaseline\nmetadata:\n  name: shared\nspec:\n  gvk:\n    version: v1\n    kind: ConfigMap\n"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	data, metadata, err := packager.PackDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	secret := packager.BuildInstallSecret("old-package", "packages", metadata, data)
	schema := runtime.NewScheme()
	if err := corev1.AddToScheme(schema); err != nil {
		t.Fatal(err)
	}
	current := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "middleware.cn/v1", "kind": "MiddlewareBaseline", "metadata": map[string]any{"name": "shared", "labels": map[string]any{"middleware.cn/packagename": "new-package"}}}}
	cli := fake.NewClientBuilder().WithScheme(schema).WithObjects(secret, current).Build()
	engine := &Engine{Client: cli, PackageNamespace: "packages"}
	owner := &unstructured.Unstructured{}
	owner.SetLabels(map[string]string{"middleware.cn/packagename": "old-package"})
	got, err := engine.sourceObject(context.Background(), "MiddlewareBaseline", "shared", owner)
	if err != nil {
		t.Fatal(err)
	}
	if stringAt(got, "spec", "gvk", "kind") != "ConfigMap" {
		t.Fatal(got)
	}
}

// Zero-annotation resources must hash identically before and after the receipt
// annotation is added and stripped; empty and absent annotations are the same.
func TestBusinessHashTreatsEmptyAnnotationsAsAbsent(t *testing.T) {
	before := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "x", "namespace": "n"}, "data": map[string]any{"k": "v"}}}
	after := before.DeepCopy()
	after.SetAnnotations(map[string]string{ReceiptAnnotation: "r"})
	if businessHash(before) != businessHash(after) {
		t.Fatalf("businessHash must ignore the receipt annotation without inventing an empty annotations object")
	}
	withOther := before.DeepCopy()
	withOther.SetAnnotations(map[string]string{"keep": "1", ReceiptAnnotation: "r"})
	if businessHash(before) == businessHash(withOther) {
		t.Fatalf("businessHash must still react to real annotations")
	}
}
