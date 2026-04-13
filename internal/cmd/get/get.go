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

// Package get implements the "saola get" top-level command and its resource sub-commands.
//
// get 包实现 "saola get" 顶层命令及各资源类型子命令。
package get

import (
	"gitee.com/opensaola/saola-cli/internal/cmd/action"
	"gitee.com/opensaola/saola-cli/internal/cmd/baseline"
	"gitee.com/opensaola/saola-cli/internal/cmd/middleware"
	"gitee.com/opensaola/saola-cli/internal/cmd/operator"
	pkgcmd "gitee.com/opensaola/saola-cli/internal/cmd/pkgcmd"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdGet returns the "saola get" parent command with all resource sub-commands attached.
//
// NewCmdGet 返回带有所有资源子命令的 "saola get" 父命令。
func NewCmdGet(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: lang.T("获取或列出资源", "Get or list resources"),
		Long: lang.T(
			`获取或列出集群中的各类资源。

支持的资源类型：
  middleware  (别名: mw)   — Middleware 资源
  operator    (别名: op)   — MiddlewareOperator 资源
  action      (别名: act)  — MiddlewareAction 资源
  baseline    (别名: bl)   — Middleware/Operator Baseline
  package     (别名: pkg)  — 已安装的中间件包
  all                      — 聚合输出 middleware、operator 和 action`,
			`Get or list resources in the cluster.

Supported resource types:
  middleware  (alias: mw)   — Middleware resources
  operator    (alias: op)   — MiddlewareOperator resources
  action      (alias: act)  — MiddlewareAction resources
  baseline    (alias: bl)   — Middleware/Operator baselines
  package     (alias: pkg)  — Installed middleware packages
  all                       — Aggregate output of middleware, operator and action`,
		),
		Example: lang.T(
			`  # 列出当前命名空间的 Middleware
  saola get middleware -n my-ns

  # 列出所有命名空间的 Operator
  saola get operator -A

  # 查看所有资源（middleware + operator + action）
  saola get all -n my-ns`,
			`  # List middlewares in a namespace
  saola get middleware -n my-ns

  # List operators across all namespaces
  saola get operator -A

  # View all resources (middleware + operator + action)
  saola get all -n my-ns`,
		),
	}

	cmd.AddCommand(
		newGetMiddleware(cfg),
		newGetOperator(cfg),
		newGetAction(cfg),
		newGetBaseline(cfg),
		newGetPackage(cfg),
		newGetAll(cfg),
	)

	return cmd
}

// newGetMiddleware returns the "get middleware" sub-command.
//
// newGetMiddleware 返回 "get middleware" 子命令。
func newGetMiddleware(cfg *config.Config) *cobra.Command {
	o := &middleware.GetOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "middleware [name]",
		Aliases: []string{"mw"},
		Short:   lang.T("列出或获取 Middleware 资源", "List or get Middleware resources"),
		Long: lang.T(
			`列出命名空间内的所有 Middleware 资源，或按名称获取单个资源。
使用 -A / --all-namespaces 跨所有命名空间列出。`,
			`List all Middleware resources in a namespace, or get a single one by name.
Use -A / --all-namespaces to list across all namespaces.`,
		),
		Example: `  saola get middleware
  saola get mw
  saola get middleware my-redis
  saola get middleware my-redis -o yaml
  saola get middleware -A -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, lang.T("跨所有命名空间列出", "List across all namespaces"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json|wide|name", "Output format: table|yaml|json|wide|name"))

	return cmd
}

// newGetOperator returns the "get operator" sub-command.
//
// newGetOperator 返回 "get operator" 子命令。
func newGetOperator(cfg *config.Config) *cobra.Command {
	o := &operator.GetOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "operator [name]",
		Aliases: []string{"op"},
		Short:   lang.T("列出或获取 MiddlewareOperator 资源", "List or get MiddlewareOperator resources"),
		Long: lang.T(
			`列出所有 MiddlewareOperator 资源，或按名称查询单个资源。
使用 -o yaml/json 可输出完整资源信息。`,
			`List all MiddlewareOperator resources or get a single one by name.
Use -o yaml/json for full resource output.`,
		),
		Example: `  saola get operator
  saola get op
  saola get operator redis-operator -n my-ns
  saola get operator -A -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, lang.T("跨所有命名空间列出", "List across all namespaces"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json|wide|name", "Output format: table|yaml|json|wide|name"))

	return cmd
}

// newGetAction returns the "get action" sub-command.
//
// newGetAction 返回 "get action" 子命令。
func newGetAction(cfg *config.Config) *cobra.Command {
	o := &action.GetOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "action [name]",
		Aliases: []string{"act"},
		Short:   lang.T("列出或获取 MiddlewareAction 资源", "List or get MiddlewareAction resources"),
		Long: lang.T(
			`列出命名空间内的所有 MiddlewareAction，或按名称获取单个资源。
使用 -A / --all-namespaces 跨所有命名空间列出。`,
			`List all MiddlewareActions in a namespace, or get a single one by name.
Use -A / --all-namespaces to list across all namespaces.`,
		),
		Example: `  saola get action
  saola get act
  saola get action my-action-1234567890
  saola get action my-action-1234567890 -o yaml
  saola get action -A`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, lang.T("跨所有命名空间列出", "List across all namespaces"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json|wide|name", "Output format: table|yaml|json|wide|name"))

	return cmd
}

// newGetBaseline returns the "get baseline" sub-command.
// When a name argument is provided it delegates to baseline.GetOptions;
// otherwise it delegates to baseline.ListOptions.
//
// newGetBaseline 返回 "get baseline" 子命令。
// 有 name 参数时使用 baseline.GetOptions；无 name 参数时使用 baseline.ListOptions。
func newGetBaseline(cfg *config.Config) *cobra.Command {
	var (
		pkg    string
		kind   string
		output string
	)

	cmd := &cobra.Command{
		Use:     "baseline [name]",
		Aliases: []string{"bl"},
		Short:   lang.T("获取或列出已安装包中的 Baseline", "Get or list baselines in an installed package"),
		Long: lang.T(
			`从已安装的中间件包中获取或列出 MiddlewareBaseline 或 MiddlewareOperatorBaseline。
提供 name 参数时获取单个 baseline；否则列出指定包内所有 baseline。
使用 --kind 指定 baseline 类型（middleware 或 operator）。`,
			`Get or list MiddlewareBaseline or MiddlewareOperatorBaseline from an installed package.
When a name argument is provided, a single baseline is fetched; otherwise all baselines in the package are listed.
Use --kind to specify the baseline type (middleware or operator).`,
		),
		Example: `  saola get baseline --package redis-v1
  saola get bl --package redis-v1 --kind operator
  saola get baseline default --package redis-v1
  saola get baseline default --package redis-v1 --kind operator -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Single baseline get.
				//
				// 获取单个 baseline。
				o := &baseline.GetOptions{
					Config:  cfg,
					Name:    args[0],
					Package: pkg,
					Kind:    kind,
					Output:  output,
				}
				return o.Run(cmd.Context())
			}
			// List all baselines in the package.
			//
			// 列出包内所有 baseline。
			o := &baseline.ListOptions{
				Config:  cfg,
				Package: pkg,
				Kind:    kind,
				Output:  output,
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&pkg, "package", "", lang.T("要查询的包名（必填）", "Package name to query (required)"))
	cmd.Flags().StringVar(&kind, "kind", "middleware", lang.T("Baseline 类型：middleware|operator|action", "Baseline kind: middleware|operator|action"))
	cmd.Flags().StringVarP(&output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	_ = cmd.MarkFlagRequired("package")

	return cmd
}

// newGetPackage returns the "get package" sub-command.
// When a name argument is provided it delegates to pkgcmd.InspectOptions;
// otherwise it delegates to pkgcmd.ListOptions.
//
// newGetPackage 返回 "get package" 子命令。
// 有 name 参数时使用 pkgcmd.InspectOptions；无 name 参数时使用 pkgcmd.ListOptions。
func newGetPackage(cfg *config.Config) *cobra.Command {
	var (
		component string
		version   string
		output    string
	)

	cmd := &cobra.Command{
		Use:     "package [name]",
		Aliases: []string{"pkg"},
		Short:   lang.T("列出或查看已安装的中间件包", "List or inspect installed middleware packages"),
		Long: lang.T(
			`列出 pkg-namespace 中所有已安装的中间件包，或查看指定包的详细内容。
提供 name 参数时查看单个包的内容（inspect）；否则列出所有包（list）。
支持按组件名和版本过滤。`,
			`List all installed middleware packages in the pkg-namespace, or inspect a specific package.
When a name argument is provided, the package contents are inspected; otherwise all packages are listed.
Supports filtering by component name and version.`,
		),
		Example: `  saola get package
  saola get pkg
  saola get package --component redis
  saola get package redis-v1
  saola get package redis-v1 -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Inspect a single package.
				//
				// 查看单个包的内容。
				o := &pkgcmd.InspectOptions{
					Config: cfg,
					Name:   args[0],
					Output: output,
				}
				return o.Run(cmd.Context())
			}
			// List all packages.
			//
			// 列出所有包。
			o := &pkgcmd.ListOptions{
				Config:    cfg,
				Component: component,
				Version:   version,
				Output:    output,
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&component, "component", "", lang.T("按组件名过滤（仅列表模式）", "Filter by component name (list mode only)"))
	cmd.Flags().StringVar(&version, "version", "", lang.T("按包版本过滤（仅列表模式）", "Filter by package version (list mode only)"))
	cmd.Flags().StringVarP(&output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))

	return cmd
}
