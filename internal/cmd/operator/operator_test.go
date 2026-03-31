package operator

import (
	"context"
	"os"
	"strings"
	"testing"

	zeusv1 "gitea.com/middleware-management/zeus-operator/api/v1"
	"gitea.com/middleware-management/saola-cli/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newScheme builds a runtime.Scheme that includes all types used in tests.
//
// newScheme 构建测试所需类型已注册的 runtime.Scheme。
func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = zeusv1.AddToScheme(s)
	return s
}

// newFakeClient returns a fake controller-runtime client pre-loaded with objs.
//
// newFakeClient 返回预置了 objs 的 fake controller-runtime client。
func newFakeClient(objs ...sigs.Object) sigs.Client {
	return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(objs...).Build()
}

// emptyCfg returns a minimal *config.Config suitable for unit tests.
//
// emptyCfg 返回适合单元测试的最小 *config.Config。
func emptyCfg() *config.Config {
	return &config.Config{}
}

// writeTempYAML writes content to a temp file and returns its path.
// The caller must remove the file when done.
//
// writeTempYAML 将内容写入临时文件并返回路径，调用方负责清理。
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "operator-test-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return f.Name()
}

// moYAML returns a minimal MiddlewareOperator YAML manifest.
//
// moYAML 返回最小化的 MiddlewareOperator YAML manifest。
func moYAML(name, namespace string) string {
	ns := ""
	if namespace != "" {
		ns = "\n  namespace: " + namespace
	}
	return `apiVersion: middleware.cn/v1
kind: MiddlewareOperator
metadata:
  name: ` + name + ns + `
spec:
  baseline: test-baseline
`
}

// ---------------------------------------------------------------------------
// Create tests
// 创建命令测试
// ---------------------------------------------------------------------------

// TestOperatorCreate_Success verifies a valid manifest is created when namespace
// is supplied via the --namespace flag (o.Namespace).
//
// Note: yaml.v3 does not read json struct tags, so metadata.namespace in YAML
// is not populated automatically. The --namespace flag is the reliable path.
//
// 验证通过 --namespace flag 提供 namespace 时 manifest 能成功创建。
// 注：yaml.v3 不识别 json struct tag，因此 YAML 里的 metadata.namespace 不会
// 自动填充，--namespace flag 是可靠的路径。
func TestOperatorCreate_Success(t *testing.T) {
	yamlContent := moYAML("redis-operator", "")
	file := writeTempYAML(t, yamlContent)
	defer os.Remove(file)

	fc := newFakeClient()
	o := &CreateOptions{
		Config:    emptyCfg(),
		Namespace: "test-ns", // supplied via flag, bypasses yaml.v3 limitation
		File:      file,
		Client:    fc,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the object was actually created in the fake store.
	//
	// 验证对象确实写入了 fake store。
	mo := &zeusv1.MiddlewareOperator{}
	if err := fc.Get(context.Background(), sigs.ObjectKey{Name: "redis-operator", Namespace: "test-ns"}, mo); err != nil {
		t.Fatalf("object not found after create: %v", err)
	}
}

// TestOperatorCreate_AlreadyExists verifies that creating a duplicate object
// returns an "already exists" error. Namespace is supplied via flag.
//
// 验证创建已存在的对象时返回 already exists 错误；namespace 通过 flag 提供。
func TestOperatorCreate_AlreadyExists(t *testing.T) {
	existing := &zeusv1.MiddlewareOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-operator", Namespace: "test-ns"},
	}
	yamlContent := moYAML("redis-operator", "")
	file := writeTempYAML(t, yamlContent)
	defer os.Remove(file)

	o := &CreateOptions{
		Config:    emptyCfg(),
		Namespace: "test-ns",
		File:      file,
		Client:    newFakeClient(existing),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for duplicate object, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

// TestOperatorCreate_FileNotFound verifies that a missing file returns a read error.
//
// 验证文件不存在时返回读取错误。
func TestOperatorCreate_FileNotFound(t *testing.T) {
	o := &CreateOptions{
		Config: emptyCfg(),
		File:   "/nonexistent/path/operator.yaml",
		Client: newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "read file") {
		t.Errorf("expected 'read file' in error, got: %v", err)
	}
}

// TestOperatorCreate_MissingNamespace verifies that a manifest without namespace
// and no config namespace returns an error.
//
// 验证 manifest 和 config 均无 namespace 时返回错误。
func TestOperatorCreate_MissingNamespace(t *testing.T) {
	// YAML with no namespace field.
	//
	// 无 namespace 字段的 YAML。
	yaml := moYAML("redis-operator", "")
	file := writeTempYAML(t, yaml)
	defer os.Remove(file)

	o := &CreateOptions{
		Config: emptyCfg(), // Namespace is ""
		File:   file,
		Client: newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected namespace error, got nil")
	}
	if !strings.Contains(err.Error(), "namespace is required") {
		t.Errorf("expected 'namespace is required' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Delete tests
// 删除命令测试
// ---------------------------------------------------------------------------

// TestOperatorDelete_Success verifies that an existing object is deleted without error.
//
// 验证存在的对象能被成功删除。
func TestOperatorDelete_Success(t *testing.T) {
	existing := &zeusv1.MiddlewareOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-operator", Namespace: "test-ns"},
	}

	o := &DeleteOptions{
		Config:    emptyCfg(),
		Name:      "redis-operator",
		Namespace: "test-ns",
		Client:    newFakeClient(existing),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestOperatorDelete_NotFound verifies that deleting a non-existent object
// prints a message and returns nil (idempotent behaviour).
//
// 验证删除不存在的对象时静默打印并返回 nil（幂等行为）。
func TestOperatorDelete_NotFound(t *testing.T) {
	o := &DeleteOptions{
		Config:    emptyCfg(),
		Name:      "ghost-operator",
		Namespace: "test-ns",
		Client:    newFakeClient(), // empty store
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected nil for not-found, got: %v", err)
	}
}

// TestOperatorDelete_MissingNamespace verifies that missing namespace returns an error.
//
// 验证缺少 namespace 时返回错误。
func TestOperatorDelete_MissingNamespace(t *testing.T) {
	o := &DeleteOptions{
		Config: emptyCfg(), // Namespace is ""
		Name:   "redis-operator",
		// Namespace field is also empty
		Client: newFakeClient(),
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected namespace error, got nil")
	}
	if !strings.Contains(err.Error(), "namespace is required") {
		t.Errorf("expected 'namespace is required' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Get tests
// 查询命令测试
// ---------------------------------------------------------------------------

// TestOperatorGet_Single verifies that a single named MiddlewareOperator is retrieved.
//
// 验证按名称查询单个 MiddlewareOperator 成功。
func TestOperatorGet_Single(t *testing.T) {
	existing := &zeusv1.MiddlewareOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-operator", Namespace: "test-ns"},
		Spec:       zeusv1.MiddlewareOperatorSpec{Baseline: "redis-baseline"},
	}

	o := &GetOptions{
		Config:    emptyCfg(),
		Name:      "redis-operator",
		Namespace: "test-ns",
		Output:    "table",
		Client:    newFakeClient(existing),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestOperatorGet_List verifies listing all MiddlewareOperators in a namespace.
//
// 验证列出 namespace 内全部 MiddlewareOperator。
func TestOperatorGet_List(t *testing.T) {
	objs := []sigs.Object{
		&zeusv1.MiddlewareOperator{
			ObjectMeta: metav1.ObjectMeta{Name: "op-a", Namespace: "test-ns"},
		},
		&zeusv1.MiddlewareOperator{
			ObjectMeta: metav1.ObjectMeta{Name: "op-b", Namespace: "test-ns"},
		},
	}

	o := &GetOptions{
		Config:    emptyCfg(),
		Namespace: "test-ns",
		Output:    "table",
		Client:    newFakeClient(objs...),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestOperatorGet_NotFound verifies that getting a non-existent named object returns an error.
//
// 验证按名称查询不存在的对象时返回错误。
func TestOperatorGet_NotFound(t *testing.T) {
	o := &GetOptions{
		Config:    emptyCfg(),
		Name:      "ghost-operator",
		Namespace: "test-ns",
		Output:    "table",
		Client:    newFakeClient(), // empty store
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for not-found object, got nil")
	}
	if !strings.Contains(err.Error(), "ghost-operator") {
		t.Errorf("expected object name in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Describe tests
// Describe 命令测试
// ---------------------------------------------------------------------------

// TestOperatorDescribe_Success verifies that describe prints output for an existing object.
//
// 验证 describe 对存在的对象能成功输出信息。
func TestOperatorDescribe_Success(t *testing.T) {
	existing := &zeusv1.MiddlewareOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-operator",
			Namespace: "test-ns",
		},
		Spec: zeusv1.MiddlewareOperatorSpec{
			Baseline:        "redis-baseline",
			PermissionScope: zeusv1.PermissionScope("Namespace"),
		},
	}

	o := &DescribeOptions{
		Config:    emptyCfg(),
		Name:      "redis-operator",
		Namespace: "test-ns",
		Client:    newFakeClient(existing),
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestOperatorDescribe_NotFound verifies that describe returns an error when the object is missing.
//
// 验证 describe 查询不存在的对象时返回错误。
func TestOperatorDescribe_NotFound(t *testing.T) {
	o := &DescribeOptions{
		Config:    emptyCfg(),
		Name:      "ghost-operator",
		Namespace: "test-ns",
		Client:    newFakeClient(), // empty store
	}

	err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for not-found object, got nil")
	}
	if !strings.Contains(err.Error(), "ghost-operator") {
		t.Errorf("expected object name in error, got: %v", err)
	}
}
