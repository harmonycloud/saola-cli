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

package uninstall

import (
	"testing"

	"gitee.com/opensaola/saola-cli/internal/config"
)

// TestNewCmdUninstall_Structure verifies that the "uninstall" command has the
// correct Use string and Example text.
//
// 验证 "uninstall" 命令具有正确的 Use 字符串和 Example 文本。
func TestNewCmdUninstall_Structure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdUninstall(cfg)

	if cmd.Use == "" {
		t.Error("command Use should not be empty")
	}
	if cmd.Example == "" {
		t.Error("uninstall command should have Example text")
	}
}

// TestNewCmdUninstall_WaitFlag verifies that the --wait flag is registered.
//
// 验证 --wait flag 已注册。
func TestNewCmdUninstall_WaitFlag(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cmd := NewCmdUninstall(cfg)

	if cmd.Flags().Lookup("wait") == nil {
		t.Error("expected --wait flag to be registered")
	}
}
