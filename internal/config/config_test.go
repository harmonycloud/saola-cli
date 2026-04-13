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

package config

import (
	"testing"

	"github.com/spf13/cobra"
)

// New() should return non-nil Config with default PkgNamespace and LogLevel.
//
// New() 应返回非空 Config，并带有默认的 PkgNamespace 和 LogLevel。
func TestNew_Defaults(t *testing.T) {
	// Clear any env vars that could interfere with defaults.
	// 清除可能影响默认值的环境变量。
	t.Setenv("KUBECONFIG", "")
	t.Setenv("SAOLA_NAMESPACE", "")
	t.Setenv("SAOLA_PKG_NAMESPACE", "")

	cfg := New()
	if cfg == nil {
		t.Fatal("New() returned nil")
	}
	if cfg.PkgNamespace != DefaultPkgNamespace {
		t.Errorf("PkgNamespace: want %q, got %q", DefaultPkgNamespace, cfg.PkgNamespace)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: want %q, got %q", "info", cfg.LogLevel)
	}
	if cfg.Kubeconfig != "" {
		t.Errorf("Kubeconfig: want empty, got %q", cfg.Kubeconfig)
	}
}

// Setting KUBECONFIG env var should override Config.Kubeconfig.
//
// 设置 KUBECONFIG 环境变量后，应覆盖 Config.Kubeconfig。
func TestNew_KubeconfigEnv(t *testing.T) {
	t.Setenv("KUBECONFIG", "/tmp/test-kubeconfig")
	t.Setenv("SAOLA_NAMESPACE", "")
	t.Setenv("SAOLA_PKG_NAMESPACE", "")

	cfg := New()
	if cfg.Kubeconfig != "/tmp/test-kubeconfig" {
		t.Errorf("Kubeconfig: want %q, got %q", "/tmp/test-kubeconfig", cfg.Kubeconfig)
	}
}

// Setting SAOLA_NAMESPACE env var should override Config.Namespace.
//
// 设置 SAOLA_NAMESPACE 环境变量后，应覆盖 Config.Namespace。
func TestNew_NamespaceEnv(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	t.Setenv("SAOLA_NAMESPACE", "my-ns")
	t.Setenv("SAOLA_PKG_NAMESPACE", "")

	cfg := New()
	if cfg.Namespace != "my-ns" {
		t.Errorf("Namespace: want %q, got %q", "my-ns", cfg.Namespace)
	}
}

// Setting SAOLA_PKG_NAMESPACE env var should override Config.PkgNamespace.
//
// 设置 SAOLA_PKG_NAMESPACE 环境变量后，应覆盖 Config.PkgNamespace。
func TestNew_PkgNamespaceEnv(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	t.Setenv("SAOLA_NAMESPACE", "")
	t.Setenv("SAOLA_PKG_NAMESPACE", "custom-pkg-ns")

	cfg := New()
	if cfg.PkgNamespace != "custom-pkg-ns" {
		t.Errorf("PkgNamespace: want %q, got %q", "custom-pkg-ns", cfg.PkgNamespace)
	}
}

// BindFlags should apply flag values that have been marked Changed.
//
// BindFlags 应将被标记为 Changed 的 flag 值应用到 Config。
func TestBindFlags_Override(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	t.Setenv("SAOLA_NAMESPACE", "")
	t.Setenv("SAOLA_PKG_NAMESPACE", "")

	cfg := New()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("kubeconfig", "", "")
	cmd.Flags().String("context", "", "")
	cmd.Flags().String("namespace", "", "")
	cmd.Flags().String("pkg-namespace", "", "")
	cmd.Flags().String("log-level", "", "")
	cmd.Flags().Bool("no-color", false, "")

	// Simulate flags being set by the user on the command line.
	// 模拟用户通过命令行设置 flag。
	if err := cmd.Flags().Set("kubeconfig", "/home/user/.kube/config"); err != nil {
		t.Fatalf("failed to set kubeconfig flag: %v", err)
	}
	if err := cmd.Flags().Set("namespace", "prod"); err != nil {
		t.Fatalf("failed to set namespace flag: %v", err)
	}
	if err := cmd.Flags().Set("log-level", "debug"); err != nil {
		t.Fatalf("failed to set log-level flag: %v", err)
	}
	if err := cmd.Flags().Set("no-color", "true"); err != nil {
		t.Fatalf("failed to set no-color flag: %v", err)
	}

	if err := cfg.BindFlags(cmd); err != nil {
		t.Fatalf("BindFlags returned error: %v", err)
	}

	if cfg.Kubeconfig != "/home/user/.kube/config" {
		t.Errorf("Kubeconfig: want %q, got %q", "/home/user/.kube/config", cfg.Kubeconfig)
	}
	if cfg.Namespace != "prod" {
		t.Errorf("Namespace: want %q, got %q", "prod", cfg.Namespace)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: want %q, got %q", "debug", cfg.LogLevel)
	}
	if !cfg.NoColor {
		t.Errorf("NoColor: want true, got false")
	}
}

// BindFlags should not modify Config fields when no flags have Changed.
//
// 未设置任何 flag 时，BindFlags 不应修改 Config 字段。
func TestBindFlags_NoChange(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	t.Setenv("SAOLA_NAMESPACE", "")
	t.Setenv("SAOLA_PKG_NAMESPACE", "")

	cfg := New()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("kubeconfig", "", "")
	cmd.Flags().String("log-level", "", "")

	// No flags Set — none are Changed.
	// 没有调用 Set，因此没有任何 flag 被标记为 Changed。
	if err := cfg.BindFlags(cmd); err != nil {
		t.Fatalf("BindFlags returned error: %v", err)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel should remain default %q, got %q", "info", cfg.LogLevel)
	}
	if cfg.Kubeconfig != "" {
		t.Errorf("Kubeconfig should remain empty, got %q", cfg.Kubeconfig)
	}
}
