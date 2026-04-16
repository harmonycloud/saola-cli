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

package version

import (
	"fmt"
	"os"

	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/harmonycloud/saola-cli/internal/printer"
	internalversion "github.com/harmonycloud/saola-cli/internal/version"
	"github.com/spf13/cobra"
)

// NewCmdVersion returns the "version" command.
//
// 返回 version 子命令。
func NewCmdVersion(_ *config.Config) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "version",
		Short: lang.T("打印 saola-cli 版本信息", "Print saola-cli version information"),
		Long: lang.T(
			`打印 saola-cli 的版本号、构建时间和 Git 提交信息。`,
			`Print the saola-cli version number, build time, and Git commit information.`,
		),
		Example: `  saola version
  saola version -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := internalversion.Get()

			switch output {
			case "json":
				p := &printer.JSONPrinter{}
				return p.Print(os.Stdout, info)
			case "yaml":
				p := &printer.YAMLPrinter{}
				return p.Print(os.Stdout, info)
			default:
				fmt.Fprintln(os.Stdout, info.String())
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", lang.T("输出格式：json|yaml（默认：人类可读格式）", "Output format: json|yaml (default: human-readable)"))
	return cmd
}
