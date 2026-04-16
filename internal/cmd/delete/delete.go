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

// Package delete implements the "saola delete" top-level command.
//
// delete 包实现 "saola delete" 顶层命令，通过资源类型子命令路由到各类资源的删除逻辑。
package delete

import (
	"fmt"
	"os"
	"time"

	"github.com/harmonycloud/saola-cli/internal/cmd/middleware"
	"github.com/harmonycloud/saola-cli/internal/cmd/operator"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdDelete returns the "saola delete" parent command with resource type sub-commands.
//
// NewCmdDelete 返回带有资源类型子命令的 "saola delete" 父命令。
func NewCmdDelete(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: lang.T("删除资源", "Delete a resource"),
		Long: lang.T(
			`按资源类型和名称删除集群中的资源。

支持的资源类型：
  middleware  (别名: mw)  — 删除 Middleware 资源
  operator    (别名: op)  — 删除 MiddlewareOperator 资源`,
			`Delete a resource from the cluster by type and name.

Supported resource types:
  middleware  (alias: mw)  — Delete a Middleware resource
  operator    (alias: op)  — Delete a MiddlewareOperator resource`,
		),
	}

	cmd.AddCommand(
		newDeleteMiddleware(cfg),
		newDeleteOperator(cfg),
	)

	return cmd
}

// newDeleteMiddleware returns the "delete middleware" sub-command.
//
// newDeleteMiddleware 返回 "delete middleware" 子命令。
func newDeleteMiddleware(cfg *config.Config) *cobra.Command {
	var (
		namespace string
		wait      time.Duration
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:     "middleware <name>",
		Aliases: []string{"mw"},
		Short:   lang.T("删除 Middleware 资源", "Delete a Middleware resource"),
		Long: lang.T(
			`按名称删除 Middleware 资源，资源不存在时静默跳过。
使用 --wait 可等待资源完全移除。`,
			`Delete a Middleware resource by name. Silently skips if not found.
Use --wait to block until the resource is fully removed.`,
		),
		Example: `  saola delete middleware my-redis
  saola delete mw my-redis -n production
  saola delete middleware my-redis --wait 2m
  saola delete middleware my-redis --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// dry-run: print what would be deleted and return early.
			//
			// dry-run 模式：打印将要删除的资源并直接返回。
			if dryRun {
				fmt.Fprintf(os.Stdout, "middleware/%s deleted (dry-run)\n", name)
				return nil
			}

			o := &middleware.DeleteOptions{
				Config:    cfg,
				Name:      name,
				Namespace: namespace,
				Wait:      wait,
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().DurationVar(&wait, "wait", 0, lang.T("等待删除完成的超时（如 2m；0 不等待）", "Poll timeout waiting for deletion (e.g. 2m; 0 = don't wait)"))
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, lang.T("仅打印将要删除的资源，不实际执行", "Print the resource that would be deleted without actually deleting it"))

	return cmd
}

// newDeleteOperator returns the "delete operator" sub-command.
//
// newDeleteOperator 返回 "delete operator" 子命令。
func newDeleteOperator(cfg *config.Config) *cobra.Command {
	var (
		namespace string
		wait      time.Duration
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:     "operator <name>",
		Aliases: []string{"op"},
		Short:   lang.T("删除 MiddlewareOperator 资源", "Delete a MiddlewareOperator resource"),
		Long: lang.T(
			`按名称删除 MiddlewareOperator。资源不存在时静默跳过。
使用 --wait 可阻塞等待资源完全移除（Finalizer 清理可能需要一定时间）。`,
			`Delete a MiddlewareOperator by name. Silently skips if not found.
Use --wait to block until the resource is fully removed (Finalizer cleanup may take time).`,
		),
		Example: `  saola delete operator redis-operator -n my-ns
  saola delete op redis-operator -n my-ns --wait 2m
  saola delete operator redis-operator -n my-ns --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// dry-run: print what would be deleted and return early.
			//
			// dry-run 模式：打印将要删除的资源并直接返回。
			if dryRun {
				fmt.Fprintf(os.Stdout, "operator/%s deleted (dry-run)\n", name)
				return nil
			}

			o := &operator.DeleteOptions{
				Config:    cfg,
				Name:      name,
				Namespace: namespace,
				Wait:      wait,
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().DurationVar(&wait, "wait", 0, lang.T("等待删除完成的超时时间（如 2m；0 表示不等待）", "Wait for the resource to be fully deleted (e.g. 2m; 0 = don't wait)"))
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, lang.T("仅打印将要删除的资源，不实际执行", "Print the resource that would be deleted without actually deleting it"))

	return cmd
}
