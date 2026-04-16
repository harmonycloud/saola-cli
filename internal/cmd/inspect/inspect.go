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

package inspect

import (
	"github.com/harmonycloud/saola-cli/internal/cmd/pkgcmd"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdInspect returns the top-level "inspect" command.
// It is a thin wrapper around pkgcmd.InspectOptions and delegates all logic to its Run method.
//
// NewCmdInspect 返回顶层 inspect 命令。
// 作为 pkgcmd.InspectOptions 的薄封装，所有逻辑委托给其 Run 方法。
func NewCmdInspect(cfg *config.Config) *cobra.Command {
	o := &pkgcmd.InspectOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "inspect <name>",
		Short: lang.T("查看已安装包的内容", "Inspect the contents of an installed package"),
		Long: lang.T(
			`从 pkg-namespace 中读取指定包的 Secret，解压 TAR 归档并展示包内文件列表及元数据。`,
			`Read the package Secret from the pkg-namespace, decompress the TAR archive, and display the file listing and metadata.`,
		),
		Example: `  saola inspect redis-v1
  saola inspect redis-v1 -o yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))

	return cmd
}
