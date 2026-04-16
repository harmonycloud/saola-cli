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

// Package create implements the "saola create" top-level command.
//
// create 包实现 "saola create" 顶层命令，通过 YAML 文件路由到对应资源类型的创建逻辑。
package create

import (
	"fmt"
	"os"

	"github.com/harmonycloud/saola-cli/internal/client"
	"github.com/harmonycloud/saola-cli/internal/cmd/middleware"
	"github.com/harmonycloud/saola-cli/internal/cmd/operator"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	sigsyaml "sigs.k8s.io/yaml"
)

// kindMeta is a minimal struct used to extract the "kind" field from a Kubernetes manifest.
//
// kindMeta 是用于从 Kubernetes manifest 中提取 "kind" 字段的最小结构体。
type kindMeta struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

// NewCmdCreate returns the "saola create" top-level command.
// It reads a YAML file and routes to the appropriate resource type handler based on the kind field.
//
// NewCmdCreate 返回 "saola create" 顶层命令。
// 读取 YAML 文件后根据 kind 字段路由到对应资源类型的处理逻辑。
func NewCmdCreate(cfg *config.Config) *cobra.Command {
	var (
		file      string
		namespace string
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: lang.T("从 YAML 文件创建资源", "Create a resource from a YAML file"),
		Long: lang.T(
			`从 YAML 文件创建资源，或不带 -f 进入交互式创建模式。
交互模式下会引导选择 baseline、填写参数，自动生成资源清单。
支持的资源类型（kind）：Middleware、MiddlewareOperator。`,
			`Create a resource from a YAML file, or enter interactive mode without -f.
In interactive mode, you'll be guided to select a baseline, fill in parameters, and auto-generate the manifest.
Supported resource types (kind): Middleware, MiddlewareOperator.`,
		),
		Example: `  # 交互式创建 / Interactive creation
  saola create

  # 从文件创建 / Create from a file
  saola create -f middleware.yaml
  saola create -f operator.yaml -n production
  saola create -f middleware.yaml --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				// Interactive mode: enter interactive creation when -f is not provided.
				//
				// 交互模式：无 -f 时进入交互式创建。
				cli, err := client.New(cfg).Get()
				if err != nil {
					return fmt.Errorf("create k8s client: %w", err)
				}
				return RunInteractive(cmd.Context(), cfg, cli)
			}

			// Read the manifest file.
			//
			// 读取 YAML 文件内容。
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read file %q: %w", file, err)
			}

			// Parse only the kind and metadata.name fields.
			//
			// 仅解析 kind 和 metadata.name 字段，用于路由和 dry-run 输出。
			var meta kindMeta
			if err = sigsyaml.Unmarshal(data, &meta); err != nil {
				return fmt.Errorf("parse manifest: %w", err)
			}

			if meta.Kind == "" {
				return fmt.Errorf("manifest is missing required field: kind")
			}

			// dry-run: print what would be created and return early.
			//
			// dry-run 模式：打印将要创建的资源并直接返回。
			if dryRun {
				fmt.Fprintf(os.Stdout, "%s/%s created (dry-run)\n", meta.Kind, meta.Metadata.Name)
				return nil
			}

			// Route to the appropriate handler based on kind.
			//
			// 根据 kind 路由到对应的处理器。
			switch meta.Kind {
			case "Middleware":
				o := &middleware.CreateOptions{
					Config:    cfg,
					File:      file,
					Namespace: namespace,
				}
				return o.Run(cmd.Context())

			case "MiddlewareOperator":
				o := &operator.CreateOptions{
					Config:    cfg,
					File:      file,
					Namespace: namespace,
				}
				return o.Run(cmd.Context())

			default:
				return fmt.Errorf("unsupported resource kind %q (supported: Middleware, MiddlewareOperator)", meta.Kind)
			}
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", lang.T("YAML 清单文件路径", "Path to a resource manifest YAML file"))
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", lang.T("覆盖清单中的命名空间", "Override the namespace from the manifest"))
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, lang.T("仅打印将要创建的资源，不实际执行（仅文件模式）", "Print the resource that would be created without actually creating it (file mode only)"))

	return cmd
}
