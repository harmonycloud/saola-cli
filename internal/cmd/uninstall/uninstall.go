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
	"fmt"

	"github.com/harmonycloud/saola-cli/internal/cmd/pkgcmd"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
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
		Use:   "uninstall <name>|package <name>",
		Short: lang.T("卸载中间件包", "Uninstall a middleware package"),
		Long: lang.T(
			`真实卸载中间件包。命令会先检查是否仍有 Middleware 或 MiddlewareOperator 引用该包；
检查通过后给包 Secret 添加清理 finalizer 并发起删除，由 OpenSaola 清理包资源后移除 finalizer。`,
			`Really uninstall a middleware package. The command first checks whether any Middleware or MiddlewareOperator still references the package;
after the check passes, it adds a cleanup finalizer to the package Secret and deletes it; OpenSaola removes the finalizer after cleaning package resources.`,
		),
		Example: `  saola uninstall redis-v1
  saola uninstall package redis-v1
  saola uninstall redis-v1 --wait 5m`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return nil
			}
			if len(args) == 2 && (args[0] == "package" || args[0] == "pkg") {
				return nil
			}
			return fmt.Errorf("expected <name> or package <name>")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			if len(args) == 2 {
				o.Name = args[1]
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待卸载完成（0 表示不等待）", "Wait for uninstallation to complete (0 = don't wait)"))

	return cmd
}
