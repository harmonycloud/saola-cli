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
	"github.com/harmonycloud/saola-cli/internal/cmd/middleware"
	"github.com/harmonycloud/saola-cli/internal/cmd/operator"
	"github.com/harmonycloud/saola-cli/internal/cmd/pkgcmd"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdUpgrade returns the top-level "upgrade" command.
// It supports two modes:
//   - Package upgrade:  saola upgrade <pkg-dir> [--name ...] [--wait ...]
//   - Instance upgrade: saola upgrade middleware <name> --to-version <v>
//     saola upgrade operator  <name> --to-version <v>
//
// NewCmdUpgrade 返回顶层 upgrade 命令，支持两种模式：
//   - 包升级：saola upgrade <pkg-dir> [--name ...] [--wait ...]
//   - 实例升级：saola upgrade middleware <name> --to-version <v>
//     saola upgrade operator  <name> --to-version <v>
func NewCmdUpgrade(cfg *config.Config) *cobra.Command {
	o := &pkgcmd.UpgradeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: lang.T("升级包或资源实例", "Upgrade a package or resource instance"),
		Long: lang.T(
			`升级中间件包（Package Secret），或升级已部署的 Middleware / MiddlewareOperator 实例到新版本。

包升级：
  saola upgrade <pkg-dir>                          替换包 Secret
  saola upgrade <pkg-dir> --wait 5m                替换并等待安装完成

实例升级：
  saola upgrade middleware <name> --to-version <v>  触发 Middleware 实例升级
  saola upgrade operator <name> --to-version <v>    触发 MiddlewareOperator 实例升级`,
			`Upgrade a middleware package (Package Secret), or upgrade a deployed Middleware / MiddlewareOperator instance to a new version.

Package upgrade:
  saola upgrade <pkg-dir>                          Replace the package Secret
  saola upgrade <pkg-dir> --wait 5m                Replace and wait for installation

Instance upgrade:
  saola upgrade middleware <name> --to-version <v>  Trigger Middleware instance upgrade
  saola upgrade operator <name> --to-version <v>    Trigger MiddlewareOperator instance upgrade`,
		),
		Example: `  # 包升级 / Package upgrade
  saola upgrade ./my-redis
  saola upgrade ./my-redis --name redis-custom --wait 5m

  # 实例升级 / Instance upgrade
  saola upgrade middleware my-redis --to-version 2.0.0 --wait 5m
  saola upgrade mw my-redis --to-version 2.0.0 --baseline redis-cluster-v2
  saola upgrade operator my-redis-op --to-version 2.0.0
  saola upgrade op my-redis-op --to-version 2.0.0 --wait 3m`,
		// When the first arg is not a known sub-command, cobra falls through
		// to this RunE, which handles the package upgrade path.
		//
		// 当第一个参数不是已知子命令时，cobra 调用此 RunE 处理包升级路径。
		Args: func(cmd *cobra.Command, args []string) error {
			// Allow sub-commands to handle their own args.
			// For the package upgrade path, we need exactly 1 arg.
			//
			// 子命令自行处理参数。包升级路径需要恰好 1 个参数。
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run(cmd.Context())
		},
	}

	// Package upgrade flags.
	//
	// 包升级相关 flag。
	cmd.Flags().StringVar(&o.Name, "name", "", lang.T("覆盖 Secret 名称（默认：<name>-<version>）", "Override the Secret name (default: <name>-<version>)"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("升级后等待安装完成的超时时间（如 5m，0 表示不等待）", "Wait for installation to complete after upgrade (e.g. 5m, 0 = don't wait)"))

	// Instance upgrade sub-commands.
	//
	// 实例升级子命令。
	cmd.AddCommand(
		middleware.NewCmdUpgrade(cfg),
		operator.NewCmdUpgrade(cfg),
	)

	return cmd
}
