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

package pkgcmd

import (
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdPackage returns the "package" sub-command group.
//
// 返回 package 子命令组。
func NewCmdPackage(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "package",
		Aliases: []string{"pkg"},
		Short: lang.T("管理中间件包", "Manage middleware packages"),
		Long: lang.T(
			`安装、卸载、升级、列出、检查并构建中间件包。`,
			`Install, uninstall, upgrade, list, inspect and build middleware packages.`,
		),
	}

	cmd.AddCommand(
		NewCmdInstall(cfg),
		NewCmdUninstall(cfg),
		NewCmdUpgrade(cfg),
		NewCmdList(cfg),
		NewCmdInspect(cfg),
		NewCmdBuild(cfg),
	)
	return cmd
}
