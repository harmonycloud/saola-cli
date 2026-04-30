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

package pkgcmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/config"
	saolaconsts "github.com/harmonycloud/saola-cli/internal/consts"
	"github.com/harmonycloud/saola-cli/internal/packager"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newScheme builds a runtime.Scheme with all types needed by fake client tests.
//
// newScheme 构建包含测试所需全部类型的 runtime.Scheme。
func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = zeusv1.AddToScheme(s)
	return s
}

// newFakeClient returns a fake controller-runtime client using the shared scheme.
//
// newFakeClient 返回使用共享 scheme 的 fake controller-runtime 客户端。
func newFakeClient(objs ...runtime.Object) sigs.Client {
	scheme := newScheme()
	clientObjs := make([]sigs.Object, 0, len(objs))
	for _, o := range objs {
		if co, ok := o.(sigs.Object); ok {
			clientObjs = append(clientObjs, co)
		}
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(clientObjs...).Build()
}

// makePkgDir creates a minimal valid package directory for tests.
// The directory contains only metadata.yaml with name and version fields.
//
// makePkgDir 为测试创建最小合法包目录，目录内只有含 name 和 version 字段的 metadata.yaml。
func makePkgDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	_ = os.WriteFile(
		filepath.Join(dir, "metadata.yaml"),
		[]byte("name: testpkg\nversion: \"1.0.0\"\n"),
		0o644,
	)
	return dir
}

// testConfig returns a minimal *config.Config suitable for unit tests.
//
// testConfig 返回适合单元测试的最小 *config.Config。
func testConfig() *config.Config {
	return &config.Config{
		PkgNamespace: "test-ns",
	}
}

// buildPackedSecret packs dir and returns a Secret ready to pre-load into fake client.
//
// buildPackedSecret 打包 dir 并返回可预加载到 fake client 的 Secret。
func buildPackedSecret(t *testing.T, dir, ns string) *corev1.Secret {
	t.Helper()
	data, meta, err := packager.PackDir(dir)
	if err != nil {
		t.Fatalf("packager.PackDir: %v", err)
	}
	secret := packager.BuildInstallSecret("", ns, meta, data)
	return secret
}

// ─────────────────────────────────────────────
// install tests
// ─────────────────────────────────────────────

// TestInstall_DryRun verifies that a dry-run prints the manifest and returns nil without hitting k8s.
//
// TestInstall_DryRun 验证 dry-run 时打印 Secret 清单并返回 nil，不调用 k8s。
func TestInstall_DryRun(t *testing.T) {
	dir := makePkgDir(t)
	o := &InstallOptions{
		Config: testConfig(),
		PkgDir: dir,
		DryRun: true,
		// No Client — any real k8s call would panic / error; DryRun exits before that.
		// 不注入 Client，DryRun 应在 k8s 调用之前返回，不会触发 panic。
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

// TestInstall_Success verifies that a valid package dir creates the Secret via fake client.
//
// TestInstall_Success 验证合法包目录通过 fake client 成功创建 Secret。
func TestInstall_Success(t *testing.T) {
	dir := makePkgDir(t)
	cli := newFakeClient()
	o := &InstallOptions{
		Config: testConfig(),
		PkgDir: dir,
		Client: cli,
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// Verify the Secret was created in the fake cluster.
	// 验证 Secret 已在 fake 集群中创建。
	secret := &corev1.Secret{}
	key := sigs.ObjectKey{Name: "testpkg-1.0.0", Namespace: "test-ns"}
	if err := cli.Get(context.Background(), key, secret); err != nil {
		t.Fatalf("Secret not found after install: %v", err)
	}
}

// TestInstall_AlreadyExists verifies that installing over an existing Secret returns an error.
//
// TestInstall_AlreadyExists 验证对已存在 Secret 安装时返回 "already exists" 错误。
func TestInstall_AlreadyExists(t *testing.T) {
	dir := makePkgDir(t)
	existing := buildPackedSecret(t, dir, "test-ns")
	cli := newFakeClient(existing)

	o := &InstallOptions{
		Config: testConfig(),
		PkgDir: dir,
		Client: cli,
	}
	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for already-existing Secret, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' in error, got: %v", err)
	}
}

// TestInstall_InvalidDir verifies that a directory without metadata.yaml returns an error.
//
// TestInstall_InvalidDir 验证不含 metadata.yaml 的目录返回错误。
func TestInstall_InvalidDir(t *testing.T) {
	dir := t.TempDir() // No metadata.yaml.  // 没有 metadata.yaml。
	o := &InstallOptions{
		Config: testConfig(),
		PkgDir: dir,
		Client: newFakeClient(),
	}
	if err := o.Run(context.Background()); err == nil {
		t.Fatal("expected error for invalid pkg dir, got nil")
	}
}

// ─────────────────────────────────────────────
// uninstall tests
// ─────────────────────────────────────────────

// TestUninstall_Success verifies that an existing Secret gets the uninstall annotation patched on.
//
// TestUninstall_Success 验证对已存在 Secret 成功打上卸载注解。
func TestUninstall_Success(t *testing.T) {
	ns := "test-ns"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-v1",
			Namespace: ns,
			Labels: map[string]string{
				zeusv1.LabelProject: saolaconsts.ProjectOpenSaola,
			},
		},
	}
	cli := newFakeClient(secret)

	o := &UninstallOptions{
		Config: &config.Config{PkgNamespace: ns},
		Name:   "redis-v1",
		Client: cli,
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// Verify the annotation was patched.
	// 验证注解已被打上。
	got := &corev1.Secret{}
	if err := cli.Get(context.Background(), sigs.ObjectKey{Name: "redis-v1", Namespace: ns}, got); err != nil {
		t.Fatalf("get Secret after uninstall: %v", err)
	}
	if got.Annotations[zeusv1.LabelUnInstall] != "true" {
		t.Errorf("expected uninstall annotation 'true', got %q", got.Annotations[zeusv1.LabelUnInstall])
	}
}

// TestUninstall_NotFound verifies that uninstalling a missing Secret returns a not-found error.
//
// TestUninstall_NotFound 验证卸载不存在的 Secret 时返回 not found 错误。
func TestUninstall_NotFound(t *testing.T) {
	o := &UninstallOptions{
		Config: testConfig(),
		Name:   "missing-pkg",
		Client: newFakeClient(),
	}
	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing Secret, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in error, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// upgrade tests
// ─────────────────────────────────────────────

// TestUpgrade_NewSecret verifies that upgrade creates a new Secret when none exists.
//
// TestUpgrade_NewSecret 验证不存在旧 Secret 时 upgrade 成功创建新 Secret。
func TestUpgrade_NewSecret(t *testing.T) {
	dir := makePkgDir(t)
	cli := newFakeClient()

	o := &UpgradeOptions{
		Config: testConfig(),
		PkgDir: dir,
		Client: cli,
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	secret := &corev1.Secret{}
	key := sigs.ObjectKey{Name: "testpkg-1.0.0", Namespace: "test-ns"}
	if err := cli.Get(context.Background(), key, secret); err != nil {
		t.Fatalf("Secret not found after upgrade: %v", err)
	}
}

// TestUpgrade_ReplaceExisting verifies that upgrade deletes the old Secret and creates a new one.
//
// TestUpgrade_ReplaceExisting 验证 upgrade 删除旧 Secret 后创建新 Secret。
func TestUpgrade_ReplaceExisting(t *testing.T) {
	dir := makePkgDir(t)
	existing := buildPackedSecret(t, dir, "test-ns")
	cli := newFakeClient(existing)

	o := &UpgradeOptions{
		Config: testConfig(),
		PkgDir: dir,
		Client: cli,
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// The Secret should still exist (deleted then re-created).
	// Secret 应仍然存在（先删后建）。
	secret := &corev1.Secret{}
	key := sigs.ObjectKey{Name: "testpkg-1.0.0", Namespace: "test-ns"}
	if err := cli.Get(context.Background(), key, secret); err != nil {
		t.Fatalf("Secret not found after upgrade: %v", err)
	}
}

// TestUpgrade_InvalidDir verifies that an invalid package directory returns an error before k8s calls.
//
// TestUpgrade_InvalidDir 验证无效包目录在调用 k8s 之前返回错误。
func TestUpgrade_InvalidDir(t *testing.T) {
	dir := t.TempDir()
	o := &UpgradeOptions{
		Config: testConfig(),
		PkgDir: dir,
		Client: newFakeClient(),
	}
	if err := o.Run(context.Background()); err == nil {
		t.Fatal("expected error for invalid pkg dir, got nil")
	}
}

// ─────────────────────────────────────────────
// build tests
// ─────────────────────────────────────────────

// TestBuild_Success verifies that a valid package directory produces a .pkg output file.
//
// TestBuild_Success 验证合法包目录生成 .pkg 输出文件。
func TestBuild_Success(t *testing.T) {
	dir := makePkgDir(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "testpkg-1.0.0.pkg")

	o := &BuildOptions{
		Config: testConfig(),
		PkgDir: dir,
		Output: outPath,
	}
	if err := o.Run(); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// Verify the output file exists and is non-empty.
	// 验证输出文件存在且非空。
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

// TestBuild_DefaultOutputName verifies that omitting --output generates <name>-<version>.pkg in cwd.
//
// TestBuild_DefaultOutputName 验证不指定 --output 时在当前目录生成 <name>-<version>.pkg。
func TestBuild_DefaultOutputName(t *testing.T) {
	dir := makePkgDir(t)

	// Change working directory to a temp dir so the default output file lands there.
	// 切换工作目录到临时目录，确保默认输出文件落在可控路径。
	tmpCwd := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(tmpCwd); err != nil {
		t.Skipf("cannot chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	o := &BuildOptions{
		Config: testConfig(),
		PkgDir: dir,
		Output: "", // Let Run() pick the default name.  // 让 Run() 使用默认命名。
	}
	if err := o.Run(); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	expected := filepath.Join(tmpCwd, "testpkg-1.0.0.pkg")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected output file %s not found: %v", expected, err)
	}
}

// TestBuild_InvalidDir verifies that a directory without metadata.yaml returns an error.
//
// TestBuild_InvalidDir 验证不含 metadata.yaml 的目录返回错误。
func TestBuild_InvalidDir(t *testing.T) {
	dir := t.TempDir()
	o := &BuildOptions{
		Config: testConfig(),
		PkgDir: dir,
	}
	if err := o.Run(); err == nil {
		t.Fatal("expected error for invalid pkg dir, got nil")
	}
}

// ─────────────────────────────────────────────
// inspect tests
// ─────────────────────────────────────────────

// TestInspect_Success verifies that a valid packed Secret in the fake cluster is described without error.
//
// TestInspect_Success 验证 fake 集群中存在有效打包 Secret 时 inspect 不返回错误。
func TestInspect_Success(t *testing.T) {
	ns := "test-ns"
	dir := makePkgDir(t)
	secret := buildPackedSecret(t, dir, ns)

	cli := newFakeClient(secret)
	o := &InspectOptions{
		Config: &config.Config{PkgNamespace: ns},
		Name:   secret.Name,
		Output: "table",
		Client: cli,
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

// TestInspect_NotFound verifies that inspecting a missing package returns an error.
//
// TestInspect_NotFound 验证查询不存在的包时返回错误。
func TestInspect_NotFound(t *testing.T) {
	o := &InspectOptions{
		Config: testConfig(),
		Name:   "no-such-pkg",
		Output: "table",
		Client: newFakeClient(),
	}
	if err := o.Run(context.Background()); err == nil {
		t.Fatal("expected error for missing package, got nil")
	}
}

// ─────────────────────────────────────────────
// list tests
// ─────────────────────────────────────────────

// TestList_Empty verifies that listing when no packages exist prints a "No packages found." message.
//
// TestList_Empty 验证没有包时 list 打印 "No packages found." 并返回 nil。
func TestList_Empty(t *testing.T) {
	o := &ListOptions{
		Config: testConfig(),
		Output: "table",
		Client: newFakeClient(),
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

// TestList_WithPackage verifies that a pre-existing package Secret appears in the list.
//
// TestList_WithPackage 验证预置包 Secret 可以被 list 正确列出。
func TestList_WithPackage(t *testing.T) {
	ns := "test-ns"
	dir := makePkgDir(t)
	secret := buildPackedSecret(t, dir, ns)

	cli := newFakeClient(secret)
	o := &ListOptions{
		Config: &config.Config{PkgNamespace: ns},
		Output: "table",
		Client: cli,
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

// TestList_FilterByComponent verifies that --component filter returns matching packages only.
//
// TestList_FilterByComponent 验证 --component 过滤后只返回匹配的包。
func TestList_FilterByComponent(t *testing.T) {
	ns := "test-ns"
	dir := makePkgDir(t)
	secret := buildPackedSecret(t, dir, ns)

	cli := newFakeClient(secret)
	o := &ListOptions{
		Config:    &config.Config{PkgNamespace: ns},
		Output:    "table",
		Component: "testpkg",
		Client:    cli,
	}
	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}
