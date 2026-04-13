package packager

import (
	"testing"

	"gitee.com/opensaola/opensaola/pkg/service/consts"
	"gitee.com/opensaola/opensaola/pkg/service/packages"
)

// newTestMeta is a helper that returns a minimal Metadata for tests.
//
// newTestMeta 是一个辅助函数，返回测试用的最小 Metadata。
func newTestMeta(name, version string) *Metadata {
	return &Metadata{Name: name, Version: version}
}

// TestBuildInstallSecret_Labels verifies that all expected labels are set with correct values.
//
// TestBuildInstallSecret_Labels 验证返回的 Secret 所有标签均已正确设置。
func TestBuildInstallSecret_Labels(t *testing.T) {
	meta := newTestMeta("redis", "7.0.0")
	secretName := "redis-7.0.0"
	data := []byte("fake-compressed-data")

	secret := BuildInstallSecret(secretName, "middleware-operator", meta, data)

	cases := []struct {
		key      string
		expected string
	}{
		{consts.LabelProject, consts.ProjectZeusOperator},
		{consts.LabelComponent, meta.Name},
		{consts.LabelPackageVersion, meta.Version},
		{consts.LabelPackageName, secretName},
		{consts.LabelEnabled, "false"},
	}

	for _, c := range cases {
		got, ok := secret.Labels[c.key]
		if !ok {
			t.Errorf("label %q is missing", c.key)
			continue
		}
		if got != c.expected {
			t.Errorf("label %q: got %q, want %q", c.key, got, c.expected)
		}
	}
}

// TestBuildInstallSecret_InstallAnnotation verifies that the install annotation is present.
//
// TestBuildInstallSecret_InstallAnnotation 验证 Annotations 中包含 install 注解。
func TestBuildInstallSecret_InstallAnnotation(t *testing.T) {
	meta := newTestMeta("kafka", "3.6.0")
	secret := BuildInstallSecret("", "default", meta, []byte("data"))

	if secret.Annotations == nil {
		t.Fatal("Annotations is nil, expected install annotation to be set")
	}
	if val, ok := secret.Annotations[consts.LabelInstall]; !ok || val != "true" {
		t.Errorf("expected Annotations[%q]=%q, got %q (present=%v)",
			consts.LabelInstall, "true", val, ok)
	}
}

// TestBuildInstallSecret_DataKey verifies that Data uses packages.Release as the key.
//
// TestBuildInstallSecret_DataKey 验证 Data 使用 packages.Release 作为 key。
func TestBuildInstallSecret_DataKey(t *testing.T) {
	meta := newTestMeta("mysql", "8.0.0")
	payload := []byte("compressed-tar-content")
	secret := BuildInstallSecret("mysql-8.0.0", "ops", meta, payload)

	if _, ok := secret.Data[packages.Release]; !ok {
		t.Errorf("expected Data key %q to exist", packages.Release)
	}
	if string(secret.Data[packages.Release]) != string(payload) {
		t.Errorf("Data[%q] mismatch: got %q, want %q",
			packages.Release, secret.Data[packages.Release], payload)
	}
}

// TestBuildInstallSecret_Immutable verifies that Immutable is set to true.
//
// TestBuildInstallSecret_Immutable 验证 Immutable 字段被设置为 true。
func TestBuildInstallSecret_Immutable(t *testing.T) {
	meta := newTestMeta("postgres", "15.0")
	secret := BuildInstallSecret("", "pg-ns", meta, nil)

	if secret.Immutable == nil {
		t.Fatal("Immutable is nil, expected true")
	}
	if !*secret.Immutable {
		t.Error("expected Immutable=true, got false")
	}
}

// TestBuildInstallSecret_Namespace verifies that the namespace is set correctly.
//
// TestBuildInstallSecret_Namespace 验证 Namespace 字段被正确设置。
func TestBuildInstallSecret_Namespace(t *testing.T) {
	meta := newTestMeta("mongo", "6.0.0")
	ns := "middleware-operator"
	secret := BuildInstallSecret("", ns, meta, nil)

	if secret.Namespace != ns {
		t.Errorf("Namespace: got %q, want %q", secret.Namespace, ns)
	}
}

// TestBuildInstallSecret_DefaultName verifies that when name is empty, the name defaults
// to "<meta.Name>-<meta.Version>".
//
// TestBuildInstallSecret_DefaultName 验证 name 为空时，Secret 名称默认为 "<meta.Name>-<meta.Version>"。
func TestBuildInstallSecret_DefaultName(t *testing.T) {
	meta := newTestMeta("etcd", "3.5.0")
	secret := BuildInstallSecret("", "ns", meta, nil)

	expected := "etcd-3.5.0"
	if secret.Name != expected {
		t.Errorf("Name: got %q, want %q", secret.Name, expected)
	}
}

// TestBuildInstallSecret_ExplicitName verifies that an explicit name overrides the default.
//
// TestBuildInstallSecret_ExplicitName 验证显式传入 name 时覆盖默认值。
func TestBuildInstallSecret_ExplicitName(t *testing.T) {
	meta := newTestMeta("etcd", "3.5.0")
	explicit := "my-custom-name"
	secret := BuildInstallSecret(explicit, "ns", meta, nil)

	if secret.Name != explicit {
		t.Errorf("Name: got %q, want %q", secret.Name, explicit)
	}
	// LabelPackageName must also match the explicit name.
	//
	// LabelPackageName 也应与显式名称一致。
	if secret.Labels[consts.LabelPackageName] != explicit {
		t.Errorf("LabelPackageName: got %q, want %q", secret.Labels[consts.LabelPackageName], explicit)
	}
}

// TestBuildInstallSecret_EmptyData verifies that nil data is stored without panic.
//
// TestBuildInstallSecret_EmptyData 验证传入 nil data 时不会 panic，Data key 仍存在。
func TestBuildInstallSecret_EmptyData(t *testing.T) {
	meta := newTestMeta("zk", "3.9.0")
	secret := BuildInstallSecret("", "ns", meta, nil)

	if _, ok := secret.Data[packages.Release]; !ok {
		t.Errorf("expected Data key %q to exist even for nil data", packages.Release)
	}
}
