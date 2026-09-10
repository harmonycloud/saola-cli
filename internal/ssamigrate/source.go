// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package ssamigrate

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/packages"
	"github.com/harmonycloud/saola-cli/internal/tarutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (e *Engine) sourceObject(ctx context.Context, kind, name string, owner *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if name == "" {
		return nil, fmt.Errorf("%s reference missing", kind)
	}
	o := &unstructured.Unstructured{}
	o.SetAPIVersion("middleware.cn/v1")
	o.SetKind(kind)

	err := e.Client.Get(ctx, client.ObjectKey{Name: name}, o)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	pkg := owner.GetLabels()[zeusv1.LabelPackageName]
	if err == nil && o.GetDeletionTimestamp() == nil && (pkg == "" || o.GetLabels()[zeusv1.LabelPackageName] == pkg) {
		return o, nil
	}
	if pkg == "" {
		return nil, fmt.Errorf("%s is missing/deleting and owner has no package identity", kind)
	}
	return e.packageSource(ctx, pkg, kind, name)
}

func (e *Engine) packageSource(ctx context.Context, pkg, kind, name string) (*unstructured.Unstructured, error) {
	ns := e.PackageNamespace
	if ns == "" {
		ns = "middleware-operator"
	}
	secret := &unstructured.Unstructured{}
	secret.SetAPIVersion("v1")
	secret.SetKind("Secret")
	if err := e.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: pkg}, secret); err != nil {
		return nil, fmt.Errorf("read owner package archive: %w", err)
	}
	data, _, _ := unstructured.NestedStringMap(secret.Object, "data")
	compressed, err := base64.StdEncoding.DecodeString(data[packages.Release])
	if err != nil {
		return nil, err
	}
	raw, err := packages.DeCompress(compressed)
	if err != nil {
		return nil, err
	}
	archive, err := tarutil.ReadTarInfo(raw)
	if err != nil {
		return nil, err
	}
	var found *unstructured.Unstructured
	for path, content := range archive.Files {
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			continue
		}
		if !bytes.Contains(content, []byte(kind)) {
			continue
		}
		decoder := kyaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)
		for {
			obj := &unstructured.Unstructured{}
			if err := decoder.Decode(&obj.Object); err == io.EOF {
				break
			} else if err != nil {
				return nil, fmt.Errorf("decode package source %s: %w", path, err)
			}
			if obj.GetAPIVersion() == "middleware.cn/v1" && obj.GetKind() == kind && obj.GetName() == name {
				if found != nil {
					return nil, fmt.Errorf("ambiguous package source %s/%s", kind, name)
				}
				found = obj
			}
		}
	}
	if found == nil {
		return nil, fmt.Errorf("%s/%s not found in owner package %s", kind, name, pkg)
	}
	return found, nil
}

func stringAt(o *unstructured.Unstructured, path ...string) string {
	s, _, _ := unstructured.NestedString(o.Object, path...)
	return s
}
func configurationReferenced(o *unstructured.Unstructured, name string) bool {
	refs, _, _ := unstructured.NestedSlice(o.Object, "spec", "configurations")
	for _, r := range refs {
		if m, ok := r.(map[string]any); ok && m["name"] == name {
			return true
		}
	}
	return false
}

// verifySource binds the physical target to a currently referenced baseline/MCF.
// verifySource 将物理资源绑定到当前引用的 Baseline/MCF，避免只凭名称标签接管。
func (e *Engine) verifySource(ctx context.Context, owner, live *unstructured.Unstructured) error {
	kind := "MiddlewareBaseline"
	if owner.GetKind() == "MiddlewareOperator" {
		kind = "MiddlewareOperatorBaseline"
	}
	baseline, err := e.sourceObject(ctx, kind, stringAt(owner, "spec", "baseline"), owner)
	if err != nil {
		return err
	}
	configuration := live.GetAnnotations()[zeusv1.LabelConfigurations]
	if configuration != "" {
		if !configurationReferenced(owner, configuration) && !configurationReferenced(baseline, configuration) {
			return fmt.Errorf("Configuration is not statically referenced by current owner/baseline")
		}
		_, err = e.sourceObject(ctx, "MiddlewareConfiguration", configuration, owner)
		return err
	}
	if owner.GetKind() != "Middleware" || live.GetName() != owner.GetName() {
		return fmt.Errorf("not the MID primary resource")
	}
	operatorName := stringAt(owner, "spec", "operatorBaseline", "name")
	if operatorName == "" {
		operatorName = stringAt(baseline, "spec", "operatorBaseline", "name")
	}
	group, version, targetKind := "", "", ""
	if operatorName == "" {
		group = stringAt(baseline, "spec", "gvk", "group")
		version = stringAt(baseline, "spec", "gvk", "version")
		targetKind = stringAt(baseline, "spec", "gvk", "kind")
	} else {
		operator, err := e.sourceObject(ctx, "MiddlewareOperatorBaseline", operatorName, owner)
		if err != nil {
			return err
		}
		gvkName := stringAt(owner, "spec", "operatorBaseline", "gvkName")
		if gvkName == "" {
			gvkName = stringAt(baseline, "spec", "operatorBaseline", "gvkName")
		}
		gvks, _, _ := unstructured.NestedSlice(operator.Object, "spec", "gvks")
		for _, x := range gvks {
			g, ok := x.(map[string]any)
			if ok && g["name"] == gvkName {
				group, _ = g["group"].(string)
				version, _ = g["version"].(string)
				targetKind, _ = g["kind"].(string)
			}
		}
	}
	apiVersion := version
	if group != "" {
		apiVersion = group + "/" + version
	}
	if apiVersion != live.GetAPIVersion() || targetKind != live.GetKind() {
		return fmt.Errorf("resource GVK does not match current MID baseline")
	}
	return nil
}
