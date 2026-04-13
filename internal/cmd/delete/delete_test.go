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

package delete

import (
	"testing"

	"gitee.com/opensaola/saola-cli/internal/config"
)

// TestNewCmdDelete_HasSubcommands verifies that the "delete" command registers
// both "middleware" and "operator" sub-commands.
//
// 验证 "delete" 命令注册了 "middleware" 和 "operator" 子命令。
func TestNewCmdDelete_HasSubcommands(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdDelete(cfg)

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

// TestNewCmdDelete_MiddlewareAlias verifies that the "middleware" sub-command
// has the "mw" alias.
//
// 验证 "middleware" 子命令具有 "mw" 别名。
func TestNewCmdDelete_MiddlewareAlias(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdDelete(cfg)

	for _, sub := range cmd.Commands() {
		if sub.Name() == "middleware" {
			for _, alias := range sub.Aliases {
				if alias == "mw" {
					return
				}
			}
			t.Error("middleware subcommand should have alias 'mw'")
			return
		}
	}
	t.Error("middleware subcommand not found")
}

// TestNewCmdDelete_OperatorAlias verifies that the "operator" sub-command
// has the "op" alias.
//
// 验证 "operator" 子命令具有 "op" 别名。
func TestNewCmdDelete_OperatorAlias(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdDelete(cfg)

	for _, sub := range cmd.Commands() {
		if sub.Name() == "operator" {
			for _, alias := range sub.Aliases {
				if alias == "op" {
					return
				}
			}
			t.Error("operator subcommand should have alias 'op'")
			return
		}
	}
	t.Error("operator subcommand not found")
}
