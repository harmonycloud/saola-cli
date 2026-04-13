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

package inspect

import (
	"testing"

	"gitee.com/opensaola/saola-cli/internal/config"
)

// TestNewCmdInspect_Structure verifies that the "inspect" command has the
// correct Use string and Example text.
//
// 验证 "inspect" 命令具有正确的 Use 字符串和 Example 文本。
func TestNewCmdInspect_Structure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdInspect(cfg)

	if cmd.Use == "" {
		t.Error("command Use should not be empty")
	}
	if cmd.Example == "" {
		t.Error("inspect command should have Example text")
	}
}

// TestNewCmdInspect_OutputFlag verifies that the --output / -o flag is registered
// with default value "table".
//
// 验证 --output / -o flag 已注册且默认值为 "table"。
func TestNewCmdInspect_OutputFlag(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdInspect(cfg)

	f := cmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("expected --output flag to be registered")
	}
	if f.Shorthand != "o" {
		t.Errorf("expected --output shorthand to be 'o', got %q", f.Shorthand)
	}
	if f.DefValue != "table" {
		t.Errorf("expected --output default to be 'table', got %q", f.DefValue)
	}
}
