//go:build envtest

// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package ssamigrate

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestMigrationAPI(t *testing.T) {
	assets := os.Getenv("KUBEBUILDER_ASSETS")
	if assets == "" {
		t.Skip("KUBEBUILDER_ASSETS required")
	}
	testEnv := &envtest.Environment{BinaryAssetsDirectory: assets, CRDDirectoryPaths: []string{"../../../opensaola/config/crd/bases"}, ErrorIfCRDPathMissing: true}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := testEnv.Stop(); err != nil {
			t.Error(err)
		}
	}()
	ctx := context.Background()
	must := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	cli, err := client.New(cfg, client.Options{Scheme: runtime.NewScheme()})
	must(t, err)
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	must(t, err)
	raw, err := dc.RESTClient().Get().AbsPath("/openapi/v2").SetHeader("Accept", "application/json").DoRaw(ctx)
	must(t, err)
	schemas, err := ReadSchemas(raw)
	must(t, err)
	engine := &Engine{Client: cli, Discovery: dc, Current: schemas}
	owner := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "middleware.cn/v1", "kind": "Middleware", "metadata": map[string]any{"name": "ssa-test", "namespace": "default"}, "spec": map[string]any{"baseline": "unused", "configurations": []any{map[string]any{"name": "test"}}}}}
	must(t, cli.Create(ctx, owner))
	for _, obj := range []*unstructured.Unstructured{
		{Object: map[string]any{"apiVersion": "middleware.cn/v1", "kind": "MiddlewareBaseline", "metadata": map[string]any{"name": "unused"}, "spec": map[string]any{"gvk": map[string]any{"version": "v1", "kind": "ConfigMap"}}}},
		{Object: map[string]any{"apiVersion": "middleware.cn/v1", "kind": "MiddlewareConfiguration", "metadata": map[string]any{"name": "test"}, "spec": map[string]any{"template": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: unused\n"}}},
	} {
		must(t, cli.Create(ctx, obj))
	}

	create := func(t *testing.T, name string) (*unstructured.Unstructured, AuditEvent) {
		obj := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": name, "namespace": "default", "annotations": map[string]any{"middleware.cn/configurations": "test"}}, "data": map[string]any{"keep": "yes", "legacy": "old", "changed": "old", "shared": "same"}}}
		controller := true
		obj.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: owner.GetAPIVersion(), Kind: owner.GetKind(), Name: owner.GetName(), UID: owner.GetUID(), Controller: &controller}})
		req := obj.DeepCopy()
		must(t, cli.Create(ctx, obj, client.FieldOwner("manager")))
		ev := AuditEvent{AuditID: name, Stage: "ResponseComplete", Verb: "create", UserAgent: "manager/v1.1.1", RequestObject: req, ResponseObject: obj.DeepCopy()}
		ev.User.Username = "system:serviceaccount:test:opensaola"
		ev.ResponseStatus.Code = 201
		return obj, ev
	}
	t.Run("subset preserves same-manager additions shared ownership and status", func(t *testing.T) {
		obj, audit := create(t, "subset")
		copy := obj.DeepCopy()
		copy.Object["data"].(map[string]any)["manual"] = "human"
		copy.Object["data"].(map[string]any)["changed"] = "human"
		must(t, cli.Update(ctx, copy, client.FieldOwner("manager")))
		ext := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": obj.GetName(), "namespace": "default"}, "data": map[string]any{"shared": "same", "external": "human"}}}
		must(t, cli.Patch(ctx, ext, client.Apply, client.FieldOwner("human")))
		must(t, cli.Get(ctx, client.ObjectKeyFromObject(obj), obj))
		item := Inspect(obj, owner, []AuditEvent{audit}, audit.User.Username, schemas, schemas)
		if item.Status != "ready" {
			t.Fatalf("%+v", item)
		}
		for _, p := range item.Move {
			if strings.Contains(p, "manual") || strings.Contains(p, "changed") {
				t.Fatalf("unproven field moved: %s", p)
			}
		}
		uid, err := engine.ClusterUID(ctx)
		must(t, err)
		plan := &Plan{Version: 1, ClusterUID: uid, Entries: []Entry{item}}
		plan.Seal()
		dir := t.TempDir()
		must(t, os.Chmod(dir, 0700))
		_, err = engine.Apply(ctx, plan, dir)
		must(t, err)
		_, err = engine.Apply(ctx, plan, dir)
		must(t, err)
		intent := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": obj.GetName(), "namespace": "default"}, "data": map[string]any{"keep": "yes"}}}
		must(t, cli.Patch(ctx, intent, client.Apply, client.FieldOwner(ApplyManager), client.ForceOwnership))
		must(t, cli.Get(ctx, client.ObjectKeyFromObject(obj), obj))
		data, _, _ := unstructured.NestedStringMap(obj.Object, "data")
		if data["legacy"] != "" || data["manual"] != "human" || data["changed"] != "human" || data["shared"] != "same" || data["external"] != "human" {
			t.Fatal(data)
		}
		_, err = engine.Verify(ctx, plan)
		must(t, err)
	})
	t.Run("missing history and wrong writer block without mutation", func(t *testing.T) {
		obj, audit := create(t, "blocked")
		before := obj.GetResourceVersion()
		for _, writer := range []string{"", "someone-else"} {
			item := Inspect(obj, owner, []AuditEvent{audit}, writer, schemas, schemas)
			if item.Status != "blocked" {
				t.Fatal(item)
			}
		}
		must(t, cli.Get(ctx, client.ObjectKeyFromObject(obj), obj))
		if obj.GetResourceVersion() != before {
			t.Fatal("mutated")
		}
	})
	t.Run("stale plan and wrong cluster fail without writes", func(t *testing.T) {
		obj, audit := create(t, "concurrent")
		item := Inspect(obj, owner, []AuditEvent{audit}, audit.User.Username, schemas, schemas)
		uid, err := engine.ClusterUID(ctx)
		must(t, err)
		plan := &Plan{Version: 1, ClusterUID: uid, Entries: []Entry{item}}
		plan.Seal()
		must(t, cli.Patch(ctx, obj, client.RawPatch(types.MergePatchType, []byte(`{"data":{"new":"concurrent"}}`)), client.FieldOwner("human")))
		before := obj.GetResourceVersion()
		dir := t.TempDir()
		must(t, os.Chmod(dir, 0700))
		if _, err = engine.Apply(ctx, plan, dir); err == nil {
			t.Fatal("stale accepted")
		}
		must(t, cli.Get(ctx, client.ObjectKeyFromObject(obj), obj))
		if obj.GetResourceVersion() != before {
			t.Fatal("stale mutated")
		}
		plan.ClusterUID = "wrong"
		plan.Seal()
		if _, err = engine.Apply(ctx, plan, dir); err == nil {
			t.Fatal("wrong cluster accepted")
		}
	})
	t.Run("schema change blocks", func(t *testing.T) {
		obj, audit := create(t, "schema-change")
		rawCopy, _ := json.Marshal(schemas)
		other, err := ReadSchemas(rawCopy)
		must(t, err)
		for _, def := range other.Definitions {
			def.Description += "changed"
		}
		if item := Inspect(obj, owner, []AuditEvent{audit}, audit.User.Username, other, schemas); item.Status != "blocked" {
			t.Fatal(item)
		}
	})
	t.Run("lost Apply ownership is not successful migration", func(t *testing.T) {
		obj, audit := create(t, "lost-owner")
		item := Inspect(obj, owner, []AuditEvent{audit}, audit.User.Username, schemas, schemas)
		uid, err := engine.ClusterUID(ctx)
		must(t, err)
		p := &Plan{Version: 1, ClusterUID: uid, Entries: []Entry{item}}
		p.Seal()
		dir := t.TempDir()
		must(t, os.Chmod(dir, 0700))
		_, err = engine.Apply(ctx, p, dir)
		must(t, err)
		must(t, cli.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, []byte(`[{"op":"replace","path":"/metadata/managedFields","value":[{}]}]`)), client.FieldOwner("human")))
		if _, err = engine.Verify(ctx, p); err == nil {
			t.Fatal("lost ownership accepted")
		}
		if _, err = engine.Apply(ctx, p, dir); err == nil {
			t.Fatal("retry accepted lost ownership")
		}
	})
	t.Run("foreign owner fails closed", func(t *testing.T) {
		obj, audit := create(t, "foreign")
		refs := obj.GetOwnerReferences()
		refs[0].UID = "foreign-uid"
		obj.SetOwnerReferences(refs)
		if item := Inspect(obj, owner, []AuditEvent{audit}, audit.User.Username, schemas, schemas); item.Status != "blocked" {
			t.Fatal(item)
		}
	})
	t.Run("manager reuse by another principal fails closed", func(t *testing.T) {
		obj, audit := create(t, "reused")
		other := audit
		other.Verb = "patch"
		other.User.Username = "human"
		other.ResponseStatus.Code = 200
		item := Inspect(obj, owner, []AuditEvent{audit, other}, audit.User.Username, schemas, schemas)
		if item.Status != "blocked" {
			t.Fatal(item)
		}
	})

	t.Run("CRD atomic map associative list and status", func(t *testing.T) {
		crd := &unstructured.Unstructured{}
		must(t, json.Unmarshal([]byte(`{"apiVersion":"apiextensions.k8s.io/v1","kind":"CustomResourceDefinition","metadata":{"name":"migrationthings.ssa.test"},"spec":{"group":"ssa.test","scope":"Namespaced","names":{"plural":"migrationthings","singular":"migrationthing","kind":"MigrationThing"},"versions":[{"name":"v1","served":true,"storage":true,"subresources":{"status":{}},"schema":{"openAPIV3Schema":{"type":"object","properties":{"spec":{"type":"object","properties":{"atomic":{"type":"object","x-kubernetes-map-type":"atomic","additionalProperties":{"type":"string"}},"items":{"type":"array","x-kubernetes-list-type":"map","x-kubernetes-list-map-keys":["name"],"items":{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"value":{"type":"string"}}}}}},"status":{"type":"object","properties":{"phase":{"type":"string"}}}}}}}]}}`), &crd.Object))
		must(t, cli.Create(ctx, crd))
		var customSchemas *Schemas
		for attempt := 0; attempt < 100; attempt++ {
			raw, err := dc.RESTClient().Get().AbsPath("/openapi/v2").SetHeader("Accept", "application/json").DoRaw(ctx)
			if err == nil {
				customSchemas, err = ReadSchemas(raw)
				if err == nil {
					if _, err = customSchemas.Hash("ssa.test/v1", "MigrationThing"); err == nil {
						break
					}
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		if customSchemas == nil {
			t.Fatal("custom schema unavailable")
		}
		_, err := customSchemas.Hash("ssa.test/v1", "MigrationThing")
		must(t, err)
		obj := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "ssa.test/v1", "kind": "MigrationThing", "metadata": map[string]any{"name": "custom", "namespace": "default", "annotations": map[string]any{"middleware.cn/configurations": "test"}}, "spec": map[string]any{"atomic": map[string]any{"original": "one"}, "items": []any{map[string]any{"name": "old", "value": "gone"}, map[string]any{"name": "keep", "value": "yes"}}}}}
		controller := true
		obj.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: owner.GetAPIVersion(), Kind: owner.GetKind(), Name: owner.GetName(), UID: owner.GetUID(), Controller: &controller}})
		request := obj.DeepCopy()
		must(t, cli.Create(ctx, obj, client.FieldOwner("manager")))
		audit := AuditEvent{AuditID: "custom", Stage: "ResponseComplete", Verb: "create", UserAgent: "manager/v1.1.1", RequestObject: request, ResponseObject: obj.DeepCopy()}
		audit.User.Username = "system:serviceaccount:test:opensaola"
		audit.ResponseStatus.Code = 201
		must(t, unstructured.SetNestedField(obj.Object, "two", "spec", "atomic", "manual"))
		items, _, _ := unstructured.NestedSlice(obj.Object, "spec", "items")
		items = append(items, map[string]any{"name": "human", "value": "keep"})
		must(t, unstructured.SetNestedSlice(obj.Object, items, "spec", "items"))
		must(t, cli.Update(ctx, obj, client.FieldOwner("manager")))
		obj.Object["status"] = map[string]any{"phase": "Ready"}
		must(t, cli.Status().Update(ctx, obj, client.FieldOwner("status-controller")))
		item := Inspect(obj, owner, []AuditEvent{audit}, audit.User.Username, customSchemas, customSchemas)
		if item.Status != "ready" {
			t.Fatal(item.Reason)
		}
		for _, path := range item.Move {
			if strings.Contains(path, "atomic") {
				t.Fatal("partially modified atomic field moved")
			}
		}
		local := &Engine{Client: cli, Discovery: dc, Current: customSchemas}
		uid, err := local.ClusterUID(ctx)
		must(t, err)
		p := &Plan{Version: 1, ClusterUID: uid, Entries: []Entry{item}}
		p.Seal()
		dir := t.TempDir()
		must(t, os.Chmod(dir, 0700))
		_, err = local.Apply(ctx, p, dir)
		must(t, err)
		desired := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "ssa.test/v1", "kind": "MigrationThing", "metadata": map[string]any{"name": "custom", "namespace": "default"}, "spec": map[string]any{"items": []any{map[string]any{"name": "keep", "value": "yes"}}}}}
		must(t, cli.Patch(ctx, desired, client.Apply, client.FieldOwner(ApplyManager)))
		must(t, cli.Get(ctx, client.ObjectKeyFromObject(obj), obj))
		items, _, _ = unstructured.NestedSlice(obj.Object, "spec", "items")
		if len(items) != 2 {
			t.Fatal(items)
		}
		if stringAt(obj, "spec", "atomic", "manual") != "two" || stringAt(obj, "status", "phase") != "Ready" {
			t.Fatal(obj.Object)
		}
		_, err = local.Verify(ctx, p)
		must(t, err)
	})

}
