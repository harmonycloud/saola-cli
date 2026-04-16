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

package version

import (
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

// TestNewCmdVersion_Structure verifies that the "version" command has the
// correct Use string and Example text.
//
// 验证 "version" 命令具有正确的 Use 字符串和 Example 文本。
func TestNewCmdVersion_Structure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdVersion(cfg)

	if cmd.Use == "" {
		t.Error("command Use should not be empty")
	}
	if cmd.Example == "" {
		t.Error("version command should have Example text")
	}
}

// TestNewCmdVersion_OutputFlag verifies that the --output / -o flag is registered.
//
// 验证 --output / -o flag 已注册。
func TestNewCmdVersion_OutputFlag(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdVersion(cfg)

	f := cmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("expected --output flag to be registered")
	}
	if f.Shorthand != "o" {
		t.Errorf("expected --output shorthand to be 'o', got %q", f.Shorthand)
	}
}
