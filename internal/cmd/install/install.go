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

package install

import (
	"gitee.com/opensaola/saola-cli/internal/cmd/pkgcmd"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdInstall returns the top-level "install" command.
// It is a thin wrapper around pkgcmd.InstallOptions and delegates all logic to its Run method.
//
// NewCmdInstall 返回顶层 install 命令。
// 作为 pkgcmd.InstallOptions 的薄封装，所有逻辑委托给其 Run 方法。
func NewCmdInstall(cfg *config.Config) *cobra.Command {
	o := &pkgcmd.InstallOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "install <pkg-dir>",
		Short: lang.T("从本地目录安装中间件包", "Install a middleware package from a local directory"),
		Long: lang.T(
			`将本地包目录打包并在 pkg-namespace 中创建 Secret。
OpenSaola 检测到 Secret 后会自动安装该包。`,
			`Pack the local package directory and create a Secret in the pkg-namespace.
OpenSaola picks up the Secret and installs the package automatically.`,
		),
		Example: `  # 从当前目录安装，名称从 metadata.yaml 自动获取 / Install from current directory, auto-name from metadata.yaml
  saola install .

  # 指定 Secret 名称并等待最多 5 分钟 / Install with an explicit Secret name and wait up to 5 minutes
  saola install ./my-redis --name redis-v1 --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&o.Name, "name", "", lang.T("覆盖 Secret 名称（默认：<name>-<version>）", "Override the Secret name (default: <name>-<version>)"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待安装完成（如 5m，0 表示不等待）", "Wait for installation to complete (e.g. 5m, 0 = don't wait)"))
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, lang.T("打印 Secret 清单而不实际应用", "Print the Secret manifest without applying it"))

	return cmd
}
