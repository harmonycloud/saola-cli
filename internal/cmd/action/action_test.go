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

package action

import (
	"context"
	"encoding/json"
	"testing"

	zeusv1 "gitee.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ---- action run ----

// TestActionRun_Success verifies that Run creates a MiddlewareAction on the fake client.
//
// TestActionRun_Success 验证 Run 能通过 fake client 成功创建 MiddlewareAction。
func TestActionRun_Success(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()

	o := &RunOptions{
		Config:     newConfig(),
		Namespace:  "default",
		Middleware:  "my-redis",
		Baseline:   "redis-backup",
		Client:     cli,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify that exactly one MiddlewareAction was created in the fake store.
	//
	// 验证 fake store 中恰好创建了一个 MiddlewareAction。
	list := &zeusv1.MiddlewareActionList{}
	if err := cli.List(context.Background(), list); err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 MiddlewareAction, got %d", len(list.Items))
	}
	item := list.Items[0]
	if item.Spec.MiddlewareName != "my-redis" {
		t.Errorf("MiddlewareName = %q, want %q", item.Spec.MiddlewareName, "my-redis")
	}
	if item.Spec.Baseline != "redis-backup" {
		t.Errorf("Baseline = %q, want %q", item.Spec.Baseline, "redis-backup")
	}
}

// TestActionRun_MissingMiddleware checks that the cobra command rejects a missing --middleware flag.
//
// TestActionRun_MissingMiddleware 检查缺少 --middleware 时 cobra 命令返回必填 flag 错误。
func TestActionRun_MissingMiddleware(t *testing.T) {
	cfg := newConfig()
	cmd := NewCmdRun(cfg)
	// Only set --baseline; omit --middleware.
	//
	// 仅设置 --baseline，不设置 --middleware。
	cmd.SetArgs([]string{"--baseline", "redis-backup"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for missing --middleware flag, got nil")
	}
}

// ---- parseParams ----

// TestActionRun_ParseParams_Valid verifies that two key=value entries produce a correct JSON object.
//
// TestActionRun_ParseParams_Valid 验证两个 key=value 条目能生成正确的 JSON 对象。
func TestActionRun_ParseParams_Valid(t *testing.T) {
	raw, err := parseParams([]string{"key=value", "k2=v2"})
	if err != nil {
		t.Fatalf("parseParams() error: %v", err)
	}
	m := map[string]string{}
	if err := json.Unmarshal(raw.Raw, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if m["key"] != "value" {
		t.Errorf("key = %q, want %q", m["key"], "value")
	}
	if m["k2"] != "v2" {
		t.Errorf("k2 = %q, want %q", m["k2"], "v2")
	}
}

// TestActionRun_ParseParams_CommaSeparated verifies that comma-separated pairs are split correctly.
//
// TestActionRun_ParseParams_CommaSeparated 验证逗号分隔的键值对能被正确拆分。
func TestActionRun_ParseParams_CommaSeparated(t *testing.T) {
	raw, err := parseParams([]string{"k1=v1,k2=v2"})
	if err != nil {
		t.Fatalf("parseParams() error: %v", err)
	}
	m := map[string]string{}
	if err := json.Unmarshal(raw.Raw, &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if m["k1"] != "v1" {
		t.Errorf("k1 = %q, want %q", m["k1"], "v1")
	}
	if m["k2"] != "v2" {
		t.Errorf("k2 = %q, want %q", m["k2"], "v2")
	}
}

// TestActionRun_ParseParams_Invalid verifies that a param without '=' returns an error.
//
// TestActionRun_ParseParams_Invalid 验证不含 '=' 的参数会返回错误。
func TestActionRun_ParseParams_Invalid(t *testing.T) {
	_, err := parseParams([]string{"noequal"})
	if err == nil {
		t.Fatal("expected error for invalid param, got nil")
	}
}

// ---- action get ----

// TestActionGet_List verifies that GetOptions.Run lists MiddlewareActions from the fake client.
//
// TestActionGet_List 验证 GetOptions.Run 能从 fake client 列出 MiddlewareAction。
func TestActionGet_List(t *testing.T) {
	scheme := newScheme()
	existing := &zeusv1.MiddlewareAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-action-111",
			Namespace: "default",
		},
		Spec: zeusv1.MiddlewareActionSpec{
			MiddlewareName: "my-redis",
			Baseline:       "redis-backup",
		},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	o := &GetOptions{
		Config:    newConfig(),
		Namespace: "default",
		Output:    "table",
		Client:    cli,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
}

// ---- action describe ----

// TestActionDescribe_Found verifies that DescribeOptions.Run succeeds when the action exists.
//
// TestActionDescribe_Found 验证当 action 存在时 DescribeOptions.Run 能成功返回。
func TestActionDescribe_Found(t *testing.T) {
	scheme := newScheme()
	existing := &zeusv1.MiddlewareAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-action-999",
			Namespace: "default",
		},
		Spec: zeusv1.MiddlewareActionSpec{
			MiddlewareName: "my-redis",
			Baseline:       "redis-backup",
		},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	o := &DescribeOptions{
		Config:    newConfig(),
		Name:      "my-action-999",
		Namespace: "default",
		Client:    cli,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
}
