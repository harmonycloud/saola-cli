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

package run

import (
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

// TestNewCmdRun_Structure verifies that the "run" command has the correct
// Use string and Example text.
//
// 验证 "run" 命令具有正确的 Use 字符串和 Example 文本。
func TestNewCmdRun_Structure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdRun(cfg)

	if cmd.Use == "" {
		t.Error("command Use should not be empty")
	}
	if cmd.Example == "" {
		t.Error("run command should have Example text")
	}
}

// TestNewCmdRun_Flags verifies that --middleware, --namespace, --params, and
// --wait flags are registered.
//
// 验证 --middleware、--namespace、--params 和 --wait flags 已注册。
func TestNewCmdRun_Flags(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdRun(cfg)

	flags := []string{"middleware", "namespace", "params", "wait"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag to be registered", name)
		}
	}
}

// TestNewCmdRun_MiddlewareRequired verifies that the --middleware flag is
// marked as required.
//
// 验证 --middleware flag 被标记为必填。
func TestNewCmdRun_MiddlewareRequired(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdRun(cfg)

	f := cmd.Flags().Lookup("middleware")
	if f == nil {
		t.Fatal("expected --middleware flag to be registered")
	}

	// cobra stores required annotations on the flag.
	//
	// cobra 将 required 注解存储在 flag 上。
	annotations := f.Annotations
	if annotations == nil {
		t.Fatal("expected --middleware flag to have annotations (required)")
	}
	if _, ok := annotations["cobra_annotation_bash_completion_one_required_flag"]; !ok {
		t.Error("expected --middleware flag to be marked as required")
	}
}
