// Package describe implements the "saola describe" top-level command.
//
// describe 包实现 "saola describe" 顶层命令，通过资源类型子命令路由到各类资源的详情输出逻辑。
package describe

import (
	"gitee.com/opensaola/saola-cli/internal/cmd/action"
	"gitee.com/opensaola/saola-cli/internal/cmd/middleware"
	"gitee.com/opensaola/saola-cli/internal/cmd/operator"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdDescribe returns the "saola describe" parent command with resource type sub-commands.
//
// NewCmdDescribe 返回带有资源类型子命令的 "saola describe" 父命令。
func NewCmdDescribe(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: lang.T("显示资源的详细信息", "Show detailed information about a resource"),
		Long: lang.T(
			`按资源类型和名称显示集群中资源的完整详情，包含 spec、status 和 conditions。

支持的资源类型：
  middleware  (别名: mw)   — 显示 Middleware 详情
  operator    (别名: op)   — 显示 MiddlewareOperator 详情
  action      (别名: act)  — 显示 MiddlewareAction 详情`,
			`Show full details of a resource in the cluster by type and name,
including spec, status and conditions.

Supported resource types:
  middleware  (alias: mw)   — Describe a Middleware resource
  operator    (alias: op)   — Describe a MiddlewareOperator resource
  action      (alias: act)  — Describe a MiddlewareAction resource`,
		),
	}

	cmd.AddCommand(
		newDescribeMiddleware(cfg),
		newDescribeOperator(cfg),
		newDescribeAction(cfg),
	)

	return cmd
}

// newDescribeMiddleware returns the "describe middleware" sub-command.
//
// newDescribeMiddleware 返回 "describe middleware" 子命令。
func newDescribeMiddleware(cfg *config.Config) *cobra.Command {
	o := &middleware.DescribeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "middleware <name>",
		Aliases: []string{"mw"},
		Short:   lang.T("显示 Middleware 资源的详细信息", "Show detailed information about a Middleware resource"),
		Long: lang.T(
			`获取单个 Middleware 资源并以可读格式输出其 spec、status 和 conditions。`,
			`Fetch a single Middleware and print its spec, status and conditions in human-readable form.`,
		),
		Example: `  saola describe middleware my-redis
  saola describe mw my-redis -n production`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))

	return cmd
}

// newDescribeOperator returns the "describe operator" sub-command.
//
// newDescribeOperator 返回 "describe operator" 子命令。
func newDescribeOperator(cfg *config.Config) *cobra.Command {
	o := &operator.DescribeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "operator <name>",
		Aliases: []string{"op"},
		Short:   lang.T("显示 MiddlewareOperator 资源的详细信息", "Show detailed information of a MiddlewareOperator resource"),
		Long: lang.T(
			`打印 MiddlewareOperator 的完整 spec、status、conditions 及每个 operator 的 deployment 状态。`,
			`Print the full spec, status, conditions, and per-operator deployment status of a MiddlewareOperator.`,
		),
		Example: `  saola describe operator redis-operator -n my-ns
  saola describe op redis-operator -n my-ns`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))

	return cmd
}

// newDescribeAction returns the "describe action" sub-command.
//
// newDescribeAction 返回 "describe action" 子命令。
func newDescribeAction(cfg *config.Config) *cobra.Command {
	o := &action.DescribeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "action <name>",
		Aliases: []string{"act"},
		Short:   lang.T("显示 MiddlewareAction 资源的详细信息", "Show detailed information of a MiddlewareAction resource"),
		Long: lang.T(
			`按名称获取 MiddlewareAction 并以人类可读的格式展示其完整状态和事件信息。`,
			`Fetch a MiddlewareAction by name and display its full status and event details in a human-readable format.`,
		),
		Example: `  saola describe action my-action-1234567890
  saola describe act my-action-1234567890 -n my-namespace`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))

	return cmd
}
