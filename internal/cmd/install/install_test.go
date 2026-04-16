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

package install

import (
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

// TestNewCmdInstall_Structure verifies that the "install" command has the
// correct Use string and Example text.
//
// 验证 "install" 命令具有正确的 Use 字符串和 Example 文本。
func TestNewCmdInstall_Structure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdInstall(cfg)

	if cmd.Use == "" {
		t.Error("command Use should not be empty")
	}
	if cmd.Example == "" {
		t.Error("install command should have Example text")
	}
}

// TestNewCmdInstall_Flags verifies that --name, --wait, and --dry-run flags
// are registered.
//
// 验证 --name、--wait 和 --dry-run flags 已注册。
func TestNewCmdInstall_Flags(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdInstall(cfg)

	flags := []string{"name", "wait", "dry-run"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag to be registered", name)
		}
	}
}
