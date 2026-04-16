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

package upgrade

import (
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

// TestNewCmdUpgrade_HasSubcommands verifies that the "upgrade" command registers
// "middleware" and "operator" sub-commands for instance upgrades.
//
// 验证 "upgrade" 命令注册了 "middleware" 和 "operator" 子命令用于实例升级。
func TestNewCmdUpgrade_HasSubcommands(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdUpgrade(cfg)

	// Collect registered sub-command names.
	//
	// 收集已注册的子命令名称。
	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	expected := []string{"middleware", "operator"}
	for _, name := range expected {
		if !subCmds[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// TestNewCmdUpgrade_HasExample verifies that the "upgrade" command provides
// Example text.
//
// 验证 "upgrade" 命令提供了 Example 文本。
func TestNewCmdUpgrade_HasExample(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdUpgrade(cfg)

	if cmd.Example == "" {
		t.Error("upgrade command should have Example text")
	}
}

// TestNewCmdUpgrade_PackageFlags verifies that --name and --wait flags are
// registered for the package upgrade path.
//
// 验证包升级路径的 --name 和 --wait flags 已注册。
func TestNewCmdUpgrade_PackageFlags(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdUpgrade(cfg)

	flags := []string{"name", "wait"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag to be registered", name)
		}
	}
}
