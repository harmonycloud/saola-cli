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

package middleware

import (
	"context"
	"os"
	"strings"
	"testing"

	zeusv1 "github.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newScheme registers all types needed by the fake client.
//
// newScheme 注册 fake client 所需的所有类型。
func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = zeusv1.AddToScheme(s)
	return s
}

// newFakeClient builds a fake controller-runtime client with the given seed objects.
//
// newFakeClient 使用给定的初始对象构建 fake controller-runtime 客户端。
func newFakeClient(objs ...sigs.Object) sigs.Client {
	return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(objs...).Build()
}

// newCfg returns a minimal Config for tests.
//
// newCfg 返回测试用的最小 Config。
func newCfg(ns string) *config.Config {
	return &config.Config{Namespace: ns}
}

// writeTempYAML writes content to a temp file and returns its path.
// The caller is responsible for removing the file.
//
// writeTempYAML 将内容写入临时文件并返回路径，调用方负责删除。
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "mw-test-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err = f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

// ---------------------------------------------------------------------------
// create tests
// create 相关测试
// ---------------------------------------------------------------------------

// TestMiddlewareCreate_Success verifies that a valid manifest is created in the cluster.
//
// TestMiddlewareCreate_Success 验证合法 manifest 能在集群中成功创建。
func TestMiddlewareCreate_Success(t *testing.T) {
	yaml := `
apiVersion: middleware.cn/v1
kind: Middleware
metadata:
  name: my-redis
  namespace: staging
  labels:
    middleware.cn/packagename: redis-1.0.0
spec:
  baseline: redis-7
`
	path := writeTempYAML(t, yaml)
	defer os.Remove(path)

	o := &CreateOptions{
		Config: newCfg(""),
		File:   path,
		Client: newFakeClient(),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestMiddlewareCreate_FileNotFound verifies that a missing file returns an error.
//
// TestMiddlewareCreate_FileNotFound 验证文件不存在时返回错误。
func TestMiddlewareCreate_FileNotFound(t *testing.T) {
	o := &CreateOptions{
		Config: newCfg("default"),
		File:   "/non/existent/path.yaml",
		Client: newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "read file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestMiddlewareCreate_InvalidYAML verifies that malformed YAML returns a parse error.
// Uses an unclosed bracket which is invalid in both YAML and JSON contexts.
//
// TestMiddlewareCreate_InvalidYAML 验证 YAML 格式错误时返回解析错误。
// 使用未闭合的括号，在 YAML 和 JSON 上下文中均无效。
func TestMiddlewareCreate_InvalidYAML(t *testing.T) {
	path := writeTempYAML(t, "key: [unclosed")
	defer os.Remove(path)

	o := &CreateOptions{
		Config: newCfg("default"),
		File:   path,
		Client: newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "parse middleware manifest") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestMiddlewareCreate_NamespaceOverride verifies that --namespace overrides the manifest namespace.
//
// TestMiddlewareCreate_NamespaceOverride 验证 --namespace 覆盖 manifest 中的 namespace。
func TestMiddlewareCreate_NamespaceOverride(t *testing.T) {
	yaml := `
apiVersion: middleware.cn/v1
kind: Middleware
metadata:
  name: my-redis
  namespace: original-ns
  labels:
    middleware.cn/packagename: redis-1.0.0
spec:
  baseline: redis-7
`
	path := writeTempYAML(t, yaml)
	defer os.Remove(path)

	o := &CreateOptions{
		Config:    newCfg(""),
		File:      path,
		Namespace: "override-ns",
		Client:    newFakeClient(),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// delete tests
// delete 相关测试
// ---------------------------------------------------------------------------

// TestMiddlewareDelete_Success verifies that an existing object is deleted without error.
//
// TestMiddlewareDelete_Success 验证存在的对象能成功删除。
func TestMiddlewareDelete_Success(t *testing.T) {
	mw := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-redis",
			Namespace: "default",
		},
	}

	o := &DeleteOptions{
		Config:    newCfg("default"),
		Name:      "my-redis",
		Namespace: "default",
		Client:    newFakeClient(mw),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestMiddlewareDelete_NotFound verifies that a missing object is handled silently (no error).
//
// TestMiddlewareDelete_NotFound 验证对象不存在时静默跳过，不返回错误。
func TestMiddlewareDelete_NotFound(t *testing.T) {
	o := &DeleteOptions{
		Config:    newCfg("default"),
		Name:      "ghost-redis",
		Namespace: "default",
		Client:    newFakeClient(),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected silent success for NotFound, got: %v", err)
	}
}

// TestMiddlewareDelete_NamespaceFromConfig verifies that Config.Namespace is used when --namespace is omitted.
//
// TestMiddlewareDelete_NamespaceFromConfig 验证未指定 --namespace 时使用 Config.Namespace。
func TestMiddlewareDelete_NamespaceFromConfig(t *testing.T) {
	mw := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-redis",
			Namespace: "from-config",
		},
	}

	o := &DeleteOptions{
		Config: newCfg("from-config"),
		Name:   "my-redis",
		// Namespace is intentionally empty; should fall back to Config.Namespace.
		//
		// Namespace 故意留空，应回退到 Config.Namespace。
		Client: newFakeClient(mw),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// get tests
// get 相关测试
// ---------------------------------------------------------------------------

// TestMiddlewareGet_Single verifies that a named object is fetched and printed.
//
// TestMiddlewareGet_Single 验证按名称获取单个对象并输出。
func TestMiddlewareGet_Single(t *testing.T) {
	mw := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-redis",
			Namespace: "default",
		},
		Spec: zeusv1.MiddlewareSpec{Baseline: "redis-7"},
	}

	o := &GetOptions{
		Config:    newCfg("default"),
		Name:      "my-redis",
		Namespace: "default",
		Output:    "table",
		Client:    newFakeClient(mw),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestMiddlewareGet_List verifies that all objects in a namespace are listed.
//
// TestMiddlewareGet_List 验证列出 namespace 内所有对象。
func TestMiddlewareGet_List(t *testing.T) {
	mw1 := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-1", Namespace: "default"},
	}
	mw2 := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-2", Namespace: "default"},
	}

	o := &GetOptions{
		Config:    newCfg("default"),
		Output:    "table",
		Namespace: "default",
		Client:    newFakeClient(mw1, mw2),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestMiddlewareGet_NotFound verifies that querying a missing object returns an error.
//
// TestMiddlewareGet_NotFound 验证查询不存在的对象时返回错误。
func TestMiddlewareGet_NotFound(t *testing.T) {
	o := &GetOptions{
		Config:    newCfg("default"),
		Name:      "ghost",
		Namespace: "default",
		Output:    "table",
		Client:    newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing object, got nil")
	}
	if !strings.Contains(err.Error(), "get middleware") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestMiddlewareGet_AllNamespaces verifies that --all-namespaces lists across all namespaces.
//
// TestMiddlewareGet_AllNamespaces 验证 --all-namespaces 能跨命名空间列出所有对象。
func TestMiddlewareGet_AllNamespaces(t *testing.T) {
	mw1 := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-a", Namespace: "ns-a"},
	}
	mw2 := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-b", Namespace: "ns-b"},
	}

	o := &GetOptions{
		Config:        newCfg(""),
		Output:        "table",
		AllNamespaces: true,
		Client:        newFakeClient(mw1, mw2),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error with --all-namespaces, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// describe tests
// describe 相关测试
// ---------------------------------------------------------------------------

// TestMiddlewareDescribe_Success verifies that a found object is printed in human-readable form.
//
// TestMiddlewareDescribe_Success 验证存在的对象能以可读格式输出。
func TestMiddlewareDescribe_Success(t *testing.T) {
	mw := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-redis",
			Namespace: "default",
			Labels:    map[string]string{"env": "test"},
		},
		Spec: zeusv1.MiddlewareSpec{Baseline: "redis-7"},
		Status: zeusv1.MiddlewareStatus{
			State: "Running",
		},
	}

	o := &DescribeOptions{
		Config:    newCfg("default"),
		Name:      "my-redis",
		Namespace: "default",
		Client:    newFakeClient(mw),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestMiddlewareDescribe_NotFound verifies that describing a missing object returns an error.
//
// TestMiddlewareDescribe_NotFound 验证对象不存在时返回错误。
func TestMiddlewareDescribe_NotFound(t *testing.T) {
	o := &DescribeOptions{
		Config:    newCfg("default"),
		Name:      "ghost",
		Namespace: "default",
		Client:    newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing object, got nil")
	}
	if !strings.Contains(err.Error(), "get middleware") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// upgrade tests
// upgrade 相关测试
// ---------------------------------------------------------------------------

// TestMiddlewareUpgrade_Success verifies that an existing Middleware is patched
// with upgrade annotations when --to-version is supplied.
//
// TestMiddlewareUpgrade_Success 验证存在的 Middleware 在指定 --to-version 时
// 能被成功打上升级 annotation。
func TestMiddlewareUpgrade_Success(t *testing.T) {
	mw := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-redis",
			Namespace: "default",
		},
		Spec: zeusv1.MiddlewareSpec{Baseline: "redis-7"},
	}

	o := &UpgradeOptions{
		Config:    newCfg("default"),
		Name:      "my-redis",
		Namespace: "default",
		ToVersion: "7.2.1",
		Client:    newFakeClient(mw),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestMiddlewareUpgrade_NotFound verifies that upgrading a non-existent
// Middleware returns an error.
//
// TestMiddlewareUpgrade_NotFound 验证对不存在的 Middleware 执行升级时返回错误。
func TestMiddlewareUpgrade_NotFound(t *testing.T) {
	o := &UpgradeOptions{
		Config:    newCfg("default"),
		Name:      "ghost-redis",
		Namespace: "default",
		ToVersion: "7.2.1",
		Client:    newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing middleware, got nil")
	}
	if !strings.Contains(err.Error(), "get middleware") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestMiddlewareUpgrade_AlreadyInProgress verifies that triggering an upgrade
// while another upgrade is pending returns an error.
//
// TestMiddlewareUpgrade_AlreadyInProgress 验证在已有升级进行中时再次触发升级会返回错误。
func TestMiddlewareUpgrade_AlreadyInProgress(t *testing.T) {
	mw := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-redis",
			Namespace: "default",
			Annotations: map[string]string{
				zeusv1.LabelUpdate: "6.0.0",
			},
		},
		Spec: zeusv1.MiddlewareSpec{Baseline: "redis-7"},
	}

	o := &UpgradeOptions{
		Config:    newCfg("default"),
		Name:      "my-redis",
		Namespace: "default",
		ToVersion: "7.2.1",
		Client:    newFakeClient(mw),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for in-progress upgrade, got nil")
	}
	if !strings.Contains(err.Error(), "upgrade already in progress") {
		t.Errorf("unexpected error message: %v", err)
	}
}
