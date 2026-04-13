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
	"os"

	"github.com/spf13/cobra"
)

// Config holds all global CLI configuration derived from persistent flags and environment variables.
//
// Config 保存从持久化 flag 和环境变量派生的所有全局 CLI 配置。
type Config struct {
	// Kubeconfig is the path to the kubeconfig file.
	Kubeconfig string
	// Context is the kubeconfig context to use.
	Context string
	// Namespace is the target namespace for middleware resources.
	Namespace string
	// PkgNamespace is the namespace where package Secrets are stored.
	PkgNamespace string
	// LogLevel controls log verbosity (debug, info, warn, error).
	LogLevel string
	// NoColor disables colored output.
	NoColor bool
}

// DefaultPkgNamespace is the default namespace for package Secrets.
//
// DefaultPkgNamespace 是包 Secret 的默认命名空间。
const DefaultPkgNamespace = "middleware-operator"

// New creates a Config with defaults resolved from environment variables.
// Flag values are bound in root.go via BindFlags; environment variables serve as fallback.
//
// 创建一个从环境变量中解析默认值的 Config；flag 值通过 root.go 的 BindFlags 绑定，环境变量作为回退。
func New() *Config {
	cfg := &Config{
		PkgNamespace: DefaultPkgNamespace,
		LogLevel:     "info",
	}
	// Allow env overrides for CI / scripting scenarios.
	if v := os.Getenv("KUBECONFIG"); v != "" {
		cfg.Kubeconfig = v
	}
	if v := os.Getenv("SAOLA_NAMESPACE"); v != "" {
		cfg.Namespace = v
	}
	if v := os.Getenv("SAOLA_PKG_NAMESPACE"); v != "" {
		cfg.PkgNamespace = v
	}
	return cfg
}

// BindFlags wires persistent cobra flags into the Config.
// Call this inside PersistentPreRunE or after ParseFlags.
//
// 将持久化 cobra flag 绑定到 Config，在 PersistentPreRunE 或 ParseFlags 之后调用。
func (c *Config) BindFlags(cmd *cobra.Command) error {
	if f := cmd.Flags().Lookup("kubeconfig"); f != nil && f.Changed {
		c.Kubeconfig = f.Value.String()
	}
	if f := cmd.Flags().Lookup("context"); f != nil && f.Changed {
		c.Context = f.Value.String()
	}
	if f := cmd.Flags().Lookup("namespace"); f != nil && f.Changed {
		c.Namespace = f.Value.String()
	}
	if f := cmd.Flags().Lookup("pkg-namespace"); f != nil && f.Changed {
		c.PkgNamespace = f.Value.String()
	}
	if f := cmd.Flags().Lookup("log-level"); f != nil && f.Changed {
		c.LogLevel = f.Value.String()
	}
	if f := cmd.Flags().Lookup("no-color"); f != nil && f.Changed {
		c.NoColor = f.Value.String() == "true"
	}
	return nil
}
