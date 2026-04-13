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

package get

import (
	"fmt"
	"os"

	"gitee.com/opensaola/saola-cli/internal/cmd/action"
	"gitee.com/opensaola/saola-cli/internal/cmd/middleware"
	"gitee.com/opensaola/saola-cli/internal/cmd/operator"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// newGetAll returns the "get all" sub-command that aggregates middleware, operator, and action.
//
// newGetAll 返回 "get all" 子命令，聚合输出 middleware、operator 和 action 资源。
func newGetAll(cfg *config.Config) *cobra.Command {
	var (
		namespace     string
		allNamespaces bool
		output        string
	)

	cmd := &cobra.Command{
		Use:   "all",
		Short: lang.T("聚合列出 middleware、operator 和 action 资源", "List middleware, operator, and action resources together"),
		Long: lang.T(
			`依次列出集群中的 Middleware、MiddlewareOperator 和 MiddlewareAction 资源。
每种资源类型前打印分隔标题，空列表也会打印标题。`,
			`List Middleware, MiddlewareOperator, and MiddlewareAction resources in sequence.
A section header is printed before each resource type, even when the list is empty.`,
		),
		Example: `  saola get all
  saola get all -n my-ns
  saola get all -A
  saola get all -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// --- Middlewares ---
			//
			// --- Middleware 资源 ---
			fmt.Fprintln(os.Stdout, "=== Middlewares ===")
			mwOpts := &middleware.GetOptions{
				Config:        cfg,
				Namespace:     namespace,
				AllNamespaces: allNamespaces,
				Output:        output,
			}
			if err := mwOpts.Run(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "error listing middlewares: %v\n", err)
			}

			fmt.Fprintln(os.Stdout, "")

			// --- Operators ---
			//
			// --- Operator 资源 ---
			fmt.Fprintln(os.Stdout, "=== Operators ===")
			opOpts := &operator.GetOptions{
				Config:        cfg,
				Namespace:     namespace,
				AllNamespaces: allNamespaces,
				Output:        output,
			}
			if err := opOpts.Run(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "error listing operators: %v\n", err)
			}

			fmt.Fprintln(os.Stdout, "")

			// --- Actions ---
			//
			// --- Action 资源 ---
			fmt.Fprintln(os.Stdout, "=== Actions ===")
			actOpts := &action.GetOptions{
				Config:        cfg,
				Namespace:     namespace,
				AllNamespaces: allNamespaces,
				Output:        output,
			}
			if err := actOpts.Run(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "error listing actions: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, lang.T("跨所有命名空间列出", "List across all namespaces"))
	cmd.Flags().StringVarP(&output, "output", "o", "table", lang.T("输出格式：table|yaml|json|wide|name", "Output format: table|yaml|json|wide|name"))

	return cmd
}
