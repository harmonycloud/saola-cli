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
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

	expected := []string{"get", "create", "delete", "describe", "disable", "run",
		"install", "uninstall", "upgrade", "build", "inspect", "validate", "images", "package", "version"}
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

// TestNewRootCmd_RejectsSSAMigration verifies that removed commands fail before
// configuration or cluster access can begin.
//
// 验证已删除的迁移命令在配置初始化及集群访问之前报错。
func TestNewRootCmd_RejectsSSAMigration(t *testing.T) {
	t.Parallel()
	for _, verb := range []string{"plan", "apply", "verify"} {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()
			cmd := NewRootCmd()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs([]string{"migrate", "ssa", verb, "--kubeconfig", filepath.Join(t.TempDir(), "missing")})
			preRunCalled := false
			cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
				preRunCalled = true
				return fmt.Errorf("removed command reached configuration initialization")
			}
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), `unknown command "migrate"`) {
				t.Fatalf("expected unknown migration command, got %v", err)
			}
			if preRunCalled {
				t.Fatal("removed command reached PersistentPreRunE")
			}
			if output.Len() != 0 {
				t.Fatalf("removed command produced output: %s", output.String())
			}
		})
	}
}

// TestNewRootCmd_HelpAndCompletion checks the public command surface after
// removing migration, including completions queried by shell scripts.
//
// 验证删除迁移功能后的帮助及 Shell 查询补全结果。
func TestNewRootCmd_HelpAndCompletion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{name: "help", args: []string{"--help"}},
		{name: "completion", args: []string{"__complete", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := NewRootCmd()
			var output, diagnostics bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&diagnostics)
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute %v: %v", tc.args, err)
			}
			text := output.String()
			if strings.Contains(text, "migrate") {
				t.Fatalf("migration command is still advertised: %s", text)
			}
			for _, name := range []string{"build", "install", "upgrade", "images", "version"} {
				if !strings.Contains(text, name) {
					t.Errorf("missing normal command %q in output: %s", name, text)
				}
			}
		})
	}
}
