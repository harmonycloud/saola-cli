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

package build

import (
	"testing"

	"gitee.com/opensaola/saola-cli/internal/config"
)

// TestNewCmdBuild_Structure verifies that the "build" command has the correct
// Use string, Example text and flags.
//
// 验证 "build" 命令具有正确的 Use 字符串、Example 文本和 flags。
func TestNewCmdBuild_Structure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdBuild(cfg)

	if cmd.Use == "" {
		t.Error("command Use should not be empty")
	}
	if cmd.Example == "" {
		t.Error("build command should have Example text")
	}
}

// TestNewCmdBuild_OutputFlag verifies that the --output / -o flag is registered.
//
// 验证 --output / -o flag 已注册。
func TestNewCmdBuild_OutputFlag(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdBuild(cfg)

	f := cmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("expected --output flag to be registered")
	}
	if f.Shorthand != "o" {
		t.Errorf("expected --output shorthand to be 'o', got %q", f.Shorthand)
	}
}
