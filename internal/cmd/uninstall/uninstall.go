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
	"gitee.com/opensaola/saola-cli/internal/cmd/pkgcmd"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdUninstall returns the top-level "uninstall" command.
// It is a thin wrapper around pkgcmd.UninstallOptions and delegates all logic to its Run method.
//
// NewCmdUninstall 返回顶层 uninstall 命令。
// 作为 pkgcmd.UninstallOptions 的薄封装，所有逻辑委托给其 Run 方法。
func NewCmdUninstall(cfg *config.Config) *cobra.Command {
	o := &pkgcmd.UninstallOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "uninstall <name>",
		Short: lang.T("卸载中间件包", "Uninstall a middleware package"),
		Long: lang.T(
			`在包对应的 Secret 上添加卸载注解。
OpenSaola 检测到注解后会自动卸载该包。`,
			`Add the uninstall annotation to the package Secret.
OpenSaola will pick up the annotation and uninstall the package.`,
		),
		Example: `  saola uninstall redis-v1
  saola uninstall redis-v1 --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待卸载完成（0 表示不等待）", "Wait for uninstallation to complete (0 = don't wait)"))

	return cmd
}
