package waiter

import (
	"context"
	"strings"
	"testing"
	"time"

	"gitea.com/middleware-management/zeus-operator/pkg/service/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	// Speed up polling for tests so they finish in milliseconds.
	//
	// 加速测试轮询，使测试在毫秒内完成。
	pollInterval = 10 * time.Millisecond
}

// makeSecret builds a minimal corev1.Secret for use in tests.
//
// makeSecret 构建测试用的最小 corev1.Secret。
func makeSecret(name, namespace string, labels, annotations map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

// newFakeClient returns a fake controller-runtime client pre-populated with the given objects.
//
// newFakeClient 返回预填充指定对象的 fake controller-runtime 客户端。
func newFakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithObjects(objs...).Build()
}

// --- WaitForInstall tests ---

// TestWaitForInstall_AlreadyEnabled verifies that WaitForInstall returns immediately
// when the Secret already has enabled=true.
//
// TestWaitForInstall_AlreadyEnabled 验证 Secret 已是 enabled=true 时，WaitForInstall 立即成功。
func TestWaitForInstall_AlreadyEnabled(t *testing.T) {
	secret := makeSecret("pkg-v1", "ns", map[string]string{
		consts.LabelEnabled: "true",
	}, nil)
	cli := newFakeClient(secret)

	ctx := context.Background()
	if err := WaitForInstall(ctx, cli, "pkg-v1", "ns"); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

// TestWaitForInstall_InstallError verifies that WaitForInstall returns an error immediately
// when the Secret has an installError annotation set.
//
// TestWaitForInstall_InstallError 验证 installError 注解被设置时，WaitForInstall 立即返回错误。
func TestWaitForInstall_InstallError(t *testing.T) {
	secret := makeSecret("pkg-v1", "ns",
		map[string]string{consts.LabelEnabled: "false"},
		map[string]string{consts.AnnotationInstallError: "CRD apply failed"},
	)
	cli := newFakeClient(secret)

	ctx := context.Background()
	err := WaitForInstall(ctx, cli, "pkg-v1", "ns")
	if err == nil {
		t.Fatal("expected error for installError annotation, got nil")
	}
	if !strings.Contains(err.Error(), "CRD apply failed") {
		t.Errorf("error should contain the install error message, got: %v", err)
	}
}

// TestWaitForInstall_Timeout verifies that WaitForInstall returns a context deadline error
// when the Secret never becomes enabled.
//
// TestWaitForInstall_Timeout 验证 Secret 始终未变为 enabled=true 时，WaitForInstall 返回超时错误。
func TestWaitForInstall_Timeout(t *testing.T) {
	secret := makeSecret("pkg-v1", "ns", map[string]string{
		consts.LabelEnabled: "false",
	}, nil)
	cli := newFakeClient(secret)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WaitForInstall(ctx, cli, "pkg-v1", "ns")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected error to mention 'timed out', got: %v", err)
	}
}

// TestWaitForInstall_EventuallyEnabled verifies that WaitForInstall succeeds after a background
// goroutine sets enabled=true on the Secret, simulating operator behaviour.
//
// TestWaitForInstall_EventuallyEnabled 验证后台 goroutine 模拟 operator 将 enabled 改为 true 后，
// WaitForInstall 最终成功返回。
func TestWaitForInstall_EventuallyEnabled(t *testing.T) {
	secret := makeSecret("pkg-v1", "ns", map[string]string{
		consts.LabelEnabled: "false",
	}, nil)
	cli := newFakeClient(secret)

	// After a short delay, simulate the operator enabling the package.
	//
	// 短暂延迟后，模拟 operator 将包设置为 enabled。
	go func() {
		time.Sleep(40 * time.Millisecond)
		updated := makeSecret("pkg-v1", "ns", map[string]string{
			consts.LabelEnabled: "true",
		}, nil)
		_ = cli.Update(context.Background(), updated)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForInstall(ctx, cli, "pkg-v1", "ns"); err != nil {
		t.Fatalf("expected success after Secret is enabled, got: %v", err)
	}
}

// --- WaitForUninstall tests ---

// TestWaitForUninstall_NotFound verifies that WaitForUninstall returns nil immediately
// when the Secret does not exist (NotFound).
//
// TestWaitForUninstall_NotFound 验证 Secret 不存在（NotFound）时，WaitForUninstall 立即成功。
func TestWaitForUninstall_NotFound(t *testing.T) {
	cli := newFakeClient() // no objects pre-populated

	ctx := context.Background()
	if err := WaitForUninstall(ctx, cli, "pkg-v1", "ns"); err != nil {
		t.Fatalf("expected success for NotFound Secret, got: %v", err)
	}
}

// TestWaitForUninstall_EnabledFalseNoAnnotation verifies that WaitForUninstall returns nil
// when the Secret exists with enabled=false and no uninstall annotation.
//
// TestWaitForUninstall_EnabledFalseNoAnnotation 验证 Secret enabled=false 且无卸载注解时，
// WaitForUninstall 立即成功。
func TestWaitForUninstall_EnabledFalseNoAnnotation(t *testing.T) {
	secret := makeSecret("pkg-v1", "ns", map[string]string{
		consts.LabelEnabled: "false",
	}, nil)
	cli := newFakeClient(secret)

	ctx := context.Background()
	if err := WaitForUninstall(ctx, cli, "pkg-v1", "ns"); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

// TestWaitForUninstall_Timeout verifies that WaitForUninstall returns a context deadline error
// when the Secret remains enabled=true with the uninstall annotation present.
//
// TestWaitForUninstall_Timeout 验证 Secret 始终有卸载注解且 enabled=true 时，WaitForUninstall 超时。
func TestWaitForUninstall_Timeout(t *testing.T) {
	secret := makeSecret("pkg-v1", "ns",
		map[string]string{consts.LabelEnabled: "true"},
		map[string]string{consts.LabelUnInstall: "true"},
	)
	cli := newFakeClient(secret)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WaitForUninstall(ctx, cli, "pkg-v1", "ns")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected error to mention 'timed out', got: %v", err)
	}
}
