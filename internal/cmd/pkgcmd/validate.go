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
	"fmt"
	"os"

	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/harmonycloud/saola-cli/internal/packagevalidator"
	"github.com/harmonycloud/saola-cli/internal/printer"
	"github.com/spf13/cobra"
)

// ValidateOptions holds parameters for "package validate".
//
// ValidateOptions 保存 "package validate" 命令的参数。
type ValidateOptions struct {
	Config *config.Config
	PkgDir string
	Output string
}

// NewCmdValidate returns the "package validate" command.
//
// 返回 package validate 子命令。
func NewCmdValidate(cfg *config.Config) *cobra.Command {
	o := &ValidateOptions{Config: cfg, Output: "table"}

	cmd := &cobra.Command{
		Use:   "validate <pkg-dir>",
		Short: lang.T("校验本地中间件包", "Validate a local middleware package"),
		Long: lang.T(
			`校验本地包目录中的 metadata.yaml、YAML 文档、OpenSaola CR 结构以及 MiddlewareConfiguration 模板渲染结果。`,
			`Validate metadata.yaml, YAML documents, OpenSaola CR structure, and rendered MiddlewareConfiguration templates in a local package directory.`,
		),
		Example: `  saola package validate ./my-redis
  saola package validate ./my-redis -o yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	return cmd
}

func (o *ValidateOptions) Run() error {
	result, err := packagevalidator.ValidateDir(o.PkgDir)
	if err != nil {
		return err
	}
	switch o.Output {
	case "yaml", "json":
		p, err := printer.New(o.Output)
		if err != nil {
			return err
		}
		return p.Print(os.Stdout, result)
	default:
		fmt.Fprintf(os.Stdout, "Package validation passed: %d YAML files, %d documents, %d rendered templates\n",
			result.Files, result.Documents, result.Templates)
	}
	return nil
}

func validatePackageDir(dir string) error {
	if _, err := packagevalidator.ValidateDir(dir); err != nil {
		return fmt.Errorf("validate package: %w", err)
	}
	return nil
}
