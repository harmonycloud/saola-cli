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

package get

import (
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

// TestNewCmdGet_HasSubcommands verifies that the "get" command registers all
// expected resource sub-commands.
//
// 验证 "get" 命令注册了所有预期的资源子命令。
func TestNewCmdGet_HasSubcommands(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdGet(cfg)

	// Collect registered sub-command names.
	//
	// 收集已注册的子命令名称。
	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	expected := []string{"middleware", "operator", "action", "baseline", "package", "all"}
	for _, name := range expected {
		if !subCmds[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// TestNewCmdGet_HasExample verifies that the "get" command provides Example text.
//
// 验证 "get" 命令提供了 Example 文本。
func TestNewCmdGet_HasExample(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdGet(cfg)
	if cmd.Example == "" {
		t.Error("get command should have Example text")
	}
}
