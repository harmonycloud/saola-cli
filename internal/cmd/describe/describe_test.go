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

package describe

import (
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

// TestNewCmdDescribe_HasSubcommands verifies that the "describe" command registers
// "middleware", "operator" and "action" sub-commands.
//
// 验证 "describe" 命令注册了 "middleware"、"operator" 和 "action" 子命令。
func TestNewCmdDescribe_HasSubcommands(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdDescribe(cfg)

	// Collect registered sub-command names.
	//
	// 收集已注册的子命令名称。
	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	expected := []string{"middleware", "operator", "action"}
	for _, name := range expected {
		if !subCmds[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// TestNewCmdDescribe_SubcommandAliases verifies that each sub-command has the
// expected alias (mw, op, act).
//
// 验证每个子命令具有预期的别名（mw、op、act）。
func TestNewCmdDescribe_SubcommandAliases(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdDescribe(cfg)

	wantAliases := map[string]string{
		"middleware": "mw",
		"operator":   "op",
		"action":     "act",
	}

	for _, sub := range cmd.Commands() {
		wantAlias, ok := wantAliases[sub.Name()]
		if !ok {
			continue
		}
		found := false
		for _, alias := range sub.Aliases {
			if alias == wantAlias {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q should have alias %q", sub.Name(), wantAlias)
		}
	}
}
