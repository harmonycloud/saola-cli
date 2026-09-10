// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package ssamigrate

import (
	"context"
	"fmt"
	"sort"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Ref struct {
	APIVersion string    `json:"apiVersion"`
	Kind       string    `json:"kind"`
	Namespace  string    `json:"namespace"`
	Name       string    `json:"name"`
	UID        types.UID `json:"uid"`
}

func ref(o *unstructured.Unstructured) Ref {
	return Ref{o.GetAPIVersion(), o.GetKind(), o.GetNamespace(), o.GetName(), o.GetUID()}
}
func (r Ref) Object() *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(schema.FromAPIVersionAndKind(r.APIVersion, r.Kind))
	o.SetName(r.Name)
	o.SetNamespace(r.Namespace)
	return o
}

type Entry struct {
	Resource      Ref                         `json:"resource"`
	Owner         Ref                         `json:"owner"`
	Configuration string                      `json:"configuration,omitempty"`
	Status        string                      `json:"status"`
	Reason        string                      `json:"reason,omitempty"`
	AuditID       string                      `json:"auditID,omitempty"`
	SchemaHash    string                      `json:"schemaHash,omitempty"`
	ObjectHash    string                      `json:"objectHash,omitempty"`
	BeforeFields  []metav1.ManagedFieldsEntry `json:"beforeManagedFields,omitempty"`
	AfterFields   []metav1.ManagedFieldsEntry `json:"afterManagedFields,omitempty"`
	Transfers     []Transfer                  `json:"transfers,omitempty"`
	Move          []string                    `json:"move,omitempty"`
	Retain        []string                    `json:"retain,omitempty"`
	Shared        []string                    `json:"shared,omitempty"`
}
type Plan struct {
	Version    int       `json:"version"`
	ID         string    `json:"id"`
	ClusterUID types.UID `json:"clusterUID"`
	Writer     string    `json:"writer"`
	Entries    []Entry   `json:"entries"`
	Errors     []string  `json:"errors,omitempty"`
}

func (p *Plan) Seal() { p.ID = ""; p.ID = digest(p) }
func (p *Plan) Valid() bool {
	id := p.ID
	c := *p
	c.ID = ""
	return p.Version == 1 && id != "" && digest(c) == id
}
func businessHash(o *unstructured.Unstructured) string {
	c := o.DeepCopy()
	delete(c.Object, "status")
	c.SetManagedFields(nil)
	c.SetResourceVersion("")
	c.SetGeneration(0)
	c.SetCreationTimestamp(metav1.Time{})
	a := c.GetAnnotations()
	delete(a, ReceiptAnnotation)
	if len(a) == 0 {
		unstructured.RemoveNestedField(c.Object, "metadata", "annotations")
	} else {
		c.SetAnnotations(a)
	}
	return digest(c.Object)
}
func fieldsHash(f []metav1.ManagedFieldsEntry) string {
	c := append([]metav1.ManagedFieldsEntry{}, f...)
	for i := range c {
		c[i].Time = nil
	}
	sort.Slice(c, func(i, j int) bool { return digest(c[i]) < digest(c[j]) })
	return digest(c)
}

// Inspect never uses the current desired template to infer removed historical fields.
// Inspect 不以当前期望模板反推已消失的历史字段。
func Inspect(live, owner *unstructured.Unstructured, events []AuditEvent, writer string, historical, current *Schemas) Entry {
	e := Entry{Resource: ref(live), Status: "blocked", Configuration: live.GetAnnotations()[zeusv1.LabelConfigurations]}
	if owner != nil {
		e.Owner = ref(owner)
	}
	block := func(err error) Entry { e.Reason = err.Error(); return e }
	if err := verifyOwner(live, owner); err != nil {
		return block(err)
	}
	if live.GetAPIVersion() == "batch/v1" && live.GetKind() == "Job" {
		e.Status = "skipped"
		e.Reason = "one-shot Job"
		return e
	}
	if live.GetAnnotations()[ReceiptAnnotation] != "" {
		e.Status = "skipped"
		e.Reason = "migration receipt exists; use verify with the original plan"
		return e
	}
	if len(events) == 0 || historical == nil {
		return block(fmt.Errorf("historical audit and OpenAPI schema required; live manager name is not provenance"))
	}
	audit, err := creation(events, live, writer)
	if err != nil {
		return block(err)
	}
	e.AuditID = audit.AuditID
	oldHash, err := historical.Hash(live.GetAPIVersion(), live.GetKind())
	if err != nil {
		return block(err)
	}
	e.SchemaHash, err = current.Hash(live.GetAPIVersion(), live.GetKind())
	if err != nil {
		return block(err)
	}
	if oldHash != e.SchemaHash {
		return block(fmt.Errorf("historical/current schema topology differs"))
	}
	h, err := historical.ProvenFields(audit.RequestObject, audit.ResponseObject, live)
	if err != nil {
		return block(err)
	}
	var legacy *metav1.ManagedFieldsEntry
	for _, f := range live.GetManagedFields() {
		if f.Subresource == "" && f.Operation == metav1.ManagedFieldsOperationUpdate && f.Manager == audit.manager() {
			if legacy != nil || f.APIVersion != live.GetAPIVersion() {
				return block(fmt.Errorf("ambiguous/cross-version legacy workflow"))
			}
			copy := f
			legacy = &copy
		}
	}
	if legacy == nil {
		e.Status = "skipped"
		e.Reason = "audited Update workflow absent"
		return e
	}
	old, err := fieldSet(*legacy)
	if err != nil {
		return block(err)
	}
	take := old.Intersection(h)
	e.Retain = paths(old.Difference(take))
	if take.Empty() {
		e.Status = "skipped"
		e.Reason = "no proven unchanged fields remain in legacy workflow"
		return e
	}
	for _, f := range live.GetManagedFields() {
		if f.Subresource != "" || f.Manager == legacy.Manager && f.Operation == legacy.Operation || f.Manager == ApplyManager && f.Operation == metav1.ManagedFieldsOperationApply {
			continue
		}
		if f.APIVersion != live.GetAPIVersion() {
			return block(fmt.Errorf("external workflow uses another API version; cannot safely compare ownership"))
		}
		external, err := fieldSet(f)
		if err != nil {
			return block(err)
		}
		e.Shared = append(e.Shared, paths(take.Intersection(external))...)
	}
	e.Transfers = []Transfer{{Manager: legacy.Manager, Fields: rawSet(take), BeforeHash: digest(rawSet(old))}}
	e.AfterFields, err = TransferFields(live, e.Transfers)
	if err != nil {
		return block(err)
	}
	e.BeforeFields = live.GetManagedFields()
	e.ObjectHash = businessHash(live)
	e.Move = paths(take)
	e.Status = "ready"
	return e
}
func verifyOwner(live, owner *unstructured.Unstructured) error {
	if owner == nil || owner.GetUID() == "" || live.GetUID() == "" || owner.GetDeletionTimestamp() != nil || live.GetDeletionTimestamp() != nil {
		return fmt.Errorf("current MID/MO identity unavailable or deleting")
	}
	r := metav1.GetControllerOf(live)
	if r == nil || r.UID != owner.GetUID() || r.Name != owner.GetName() || r.Kind != owner.GetKind() || r.APIVersion != "middleware.cn/v1" || live.GetNamespace() != owner.GetNamespace() {
		return fmt.Errorf("strict MID/MO owner identity missing; cross-scope/foreign ownership requires additional historical binding")
	}
	return nil
}

type Engine struct {
	Client           client.Client
	Discovery        discovery.DiscoveryInterface
	Current          *Schemas
	RefreshSchemas   func(context.Context) (*Schemas, error)
	PackageNamespace string
}

func (e *Engine) ClusterUID(ctx context.Context) (types.UID, error) {
	o := &unstructured.Unstructured{}
	o.SetAPIVersion("v1")
	o.SetKind("Namespace")
	err := e.Client.Get(ctx, client.ObjectKey{Name: "kube-system"}, o)
	return o.GetUID(), err
}

// Scan discovers direct MID roots and Configuration resources, never operator-created children.
// Scan 只发现 MID 主资源和 Configuration 资源，不遍历中间件 Operator 的子资源树。
func (e *Engine) Scan(ctx context.Context, namespace, writer string, events []AuditEvent, historical *Schemas) (*Plan, error) {
	uid, err := e.ClusterUID(ctx)
	if err != nil {
		return nil, err
	}
	p := &Plan{Version: 1, ClusterUID: uid, Writer: writer, Entries: []Entry{}}
	owners := map[types.UID]*unstructured.Unstructured{}
	for _, kind := range []string{"Middleware", "MiddlewareOperator"} {
		list := &unstructured.UnstructuredList{}
		list.SetAPIVersion("middleware.cn/v1")
		list.SetKind(kind + "List")
		if err = e.Client.List(ctx, list, client.InNamespace(namespace)); err != nil {
			return nil, err
		}
		for i := range list.Items {
			o := list.Items[i].DeepCopy()
			owners[o.GetUID()] = o
		}
	}
	lists, discoveryErr := e.Discovery.ServerPreferredResources()
	if discoveryErr != nil {
		p.Errors = append(p.Errors, discoveryErr.Error())
	}
	for _, group := range lists {
		for _, r := range group.APIResources {
			if r.Kind == "" || r.Kind == "Event" || r.Kind == "Secret" && group.GroupVersion != "v1" {
				continue
			}
			listable := false
			for _, v := range r.Verbs {
				if v == "list" {
					listable = true
				}
			}
			if !listable {
				continue
			}
			gvk := schema.FromAPIVersionAndKind(group.GroupVersion, r.Kind)
			list := &unstructured.UnstructuredList{}
			list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
			opts := []client.ListOption{}
			if r.Namespaced {
				opts = append(opts, client.InNamespace(namespace))
			}
			if err = e.Client.List(ctx, list, opts...); err != nil {
				p.Errors = append(p.Errors, fmt.Sprintf("list %s: %v", gvk, err))
				continue
			}
			for i := range list.Items {
				o := &list.Items[i]
				o.SetGroupVersionKind(gvk)
				controller := metav1.GetControllerOf(o)
				var owner *unstructured.Unstructured
				if controller != nil {
					owner = owners[controller.UID]
				}
				configuration := o.GetAnnotations()[zeusv1.LabelConfigurations]
				if configuration == "" {
					if owner == nil || owner.GetKind() != "Middleware" || o.GetName() != owner.GetName() {
						continue
					}
				} else if owner == nil {
					app := o.GetLabels()[zeusv1.LabelApp]
					matched := false
					for _, candidate := range owners {
						if candidate.GetName() == app {
							matched = true
							if owner != nil {
								owner = nil
								break
							}
							owner = candidate
						}
					}
					if !matched {
						continue
					}
				}
				item := Inspect(o, owner, events, writer, historical, e.Current)
				if owner != nil && item.Status == "ready" {
					if err := e.verifySource(ctx, owner, o); err != nil {
						item.Status = "blocked"
						item.Reason = err.Error()
					}
				}
				p.Entries = append(p.Entries, item)
			}
		}
	}
	sort.Slice(p.Entries, func(i, j int) bool { return digest(p.Entries[i].Resource) < digest(p.Entries[j].Resource) })
	sort.Strings(p.Errors)
	p.Seal()
	return p, nil
}
