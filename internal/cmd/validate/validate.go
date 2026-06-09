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

package validate

import (
	"github.com/harmonycloud/saola-cli/internal/cmd/pkgcmd"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdValidate returns the top-level "validate" command.
//
// NewCmdValidate 返回顶层 validate 命令。
func NewCmdValidate(cfg *config.Config) *cobra.Command {
	o := &pkgcmd.ValidateOptions{Config: cfg, Output: "table"}

	cmd := &cobra.Command{
		Use:   "validate <pkg-dir>",
		Short: lang.T("校验本地中间件包", "Validate a local middleware package"),
		Long: lang.T(
			`校验本地包目录中的 metadata.yaml、YAML 文档、OpenSaola CR 结构以及 MiddlewareConfiguration 模板渲染结果。`,
			`Validate metadata.yaml, YAML documents, OpenSaola CR structure, and rendered MiddlewareConfiguration templates in a local package directory.`,
		),
		Example: `  saola validate ./my-redis
  saola validate ./my-redis -o yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	return cmd
}
