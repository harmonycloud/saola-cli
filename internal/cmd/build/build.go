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

package build

import (
	"gitee.com/opensaola/saola-cli/internal/cmd/pkgcmd"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdBuild returns the top-level "build" command.
// It is a thin wrapper around pkgcmd.BuildOptions and delegates all logic to its Run method.
// Note: BuildOptions.Run has no context parameter.
//
// NewCmdBuild 返回顶层 build 命令。
// 作为 pkgcmd.BuildOptions 的薄封装，所有逻辑委托给其 Run 方法。
// 注意：BuildOptions.Run 不接受 context 参数。
func NewCmdBuild(cfg *config.Config) *cobra.Command {
	o := &pkgcmd.BuildOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "build <pkg-dir>",
		Short: lang.T("从本地目录构建包归档文件", "Build a package archive from a local directory"),
		Long: lang.T(
			`将本地目录打包为 zstd 压缩的 TAR 文件，不执行安装。
适用于 CI 流水线或离线分发场景。`,
			`Pack the local directory into a zstd-compressed TAR file without installing it.
Useful for CI pipelines or offline distribution.`,
		),
		Example: `  saola build .
  saola build ./my-redis --output ./dist/redis-v1.pkg`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "", lang.T("输出文件路径（默认：当前目录下的 <name>-<version>.pkg）", "Output file path (default: <name>-<version>.pkg in current dir)"))

	return cmd
}
