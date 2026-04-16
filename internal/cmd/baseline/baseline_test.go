/*
Copyright 2025 The OpenSaola Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package baseline

import (
	"context"
	"testing"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newScheme builds a Scheme that includes the types used in tests.
//
// newScheme 构建包含测试所需类型的 Scheme。
func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = zeusv1.AddToScheme(s)
	return s
}

// newConfig returns a minimal config suitable for tests.
//
// newConfig 返回一个用于测试的最小配置。
func newConfig() *config.Config {
	return &config.Config{
		Namespace:    "default",
		PkgNamespace: "middleware-operator",
	}
}

// ---- baseline get ----

// TestBaselineGet_UnknownKind verifies that an unsupported kind returns an error without any k8s call.
//
// TestBaselineGet_UnknownKind 验证不支持的 kind 直接返回错误，不发起 k8s 请求。
func TestBaselineGet_UnknownKind(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()

	o := &GetOptions{
		Config:  newConfig(),
		Name:    "default",
		Package: "redis-v1",
		Kind:    "unknown-kind",
		Output:  "yaml",
		Client:  cli,
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
}

// TestBaselineGet_PackageNotFound verifies that a missing package Secret causes an error.
// The fake client has no objects, so the packages lookup should fail.
//
// TestBaselineGet_PackageNotFound 验证包 Secret 不存在时返回错误。
// fake client 中没有任何对象，packages 查找应当失败。
func TestBaselineGet_PackageNotFound(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()

	o := &GetOptions{
		Config:  newConfig(),
		Name:    "default",
		Package: "nonexistent-pkg",
		Kind:    "middleware",
		Output:  "yaml",
		Client:  cli,
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when package Secret does not exist, got nil")
	}
}

// ---- baseline list ----

// TestBaselineList_PackageNotFound verifies that a missing package Secret causes an error during list.
//
// TestBaselineList_PackageNotFound 验证 list 时包 Secret 不存在会返回错误。
func TestBaselineList_PackageNotFound(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()

	o := &ListOptions{
		Config:  newConfig(),
		Package: "nonexistent-pkg",
		Kind:    "middleware",
		Output:  "table",
		Client:  cli,
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when package Secret does not exist, got nil")
	}
}

// TestBaselineList_UnknownKind verifies that an unsupported kind returns an error without any k8s call.
//
// TestBaselineList_UnknownKind 验证不支持的 kind 直接返回错误，不发起 k8s 请求。
func TestBaselineList_UnknownKind(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()

	o := &ListOptions{
		Config:  newConfig(),
		Package: "redis-v1",
		Kind:    "unknown-kind",
		Output:  "table",
		Client:  cli,
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
}
