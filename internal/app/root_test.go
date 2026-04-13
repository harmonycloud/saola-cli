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

package app

import (
	"testing"
)

// TestNewRootCmd_HasAllSubcommands verifies that the root command registers
// every expected top-level sub-command.
//
// 验证根命令注册了所有预期的顶级子命令。
func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	t.Parallel()
	cmd := NewRootCmd()

	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}

	expected := []string{"get", "create", "delete", "describe", "run",
		"install", "uninstall", "upgrade", "build", "inspect", "version"}
	for _, name := range expected {
		if !subCmds[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// TestNewRootCmd_PersistentFlags verifies that the root command declares the
// expected persistent flags available to all sub-commands.
//
// 验证根命令声明了所有子命令可用的预期持久化 flag。
func TestNewRootCmd_PersistentFlags(t *testing.T) {
	t.Parallel()
	cmd := NewRootCmd()

	flags := []string{"kubeconfig", "namespace", "lang", "pkg-namespace"}
	for _, name := range flags {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing persistent flag: %s", name)
		}
	}
}
