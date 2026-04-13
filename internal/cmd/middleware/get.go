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

package middleware

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	zeusk8s "gitee.com/opensaola/saola-cli/internal/k8s"
	zeusv1 "gitee.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/printer"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOptions holds parameters for "middleware get".
//
// GetOptions 保存 middleware get 子命令的所有参数。
type GetOptions struct {
	Config        *config.Config
	Name          string
	Namespace     string
	Output        string
	AllNamespaces bool
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；为 nil 时使用 client.New(cfg)。
	Client sigs.Client
}

// middlewareRow is the table row model for a single Middleware.
//
// middlewareRow 是单个 Middleware 在 table 模式下的行结构。
type middlewareRow struct {
	NAME      string
	NAMESPACE string
	BASELINE  string
	STATE     string
	AGE       string
}

// NewCmdGet returns the middleware get command.
//
// 返回 middleware get 子命令。
func NewCmdGet(cfg *config.Config) *cobra.Command {
	o := &GetOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "get [name]",
		Short: lang.T("列出或获取 Middleware 资源", "List or get Middleware resources"),
		Long: lang.T(
			`列出命名空间内的所有 Middleware 资源，或按名称获取单个资源。
使用 -A / --all-namespaces 跨所有命名空间列出。`,
			`List all Middleware resources in a namespace, or get a single one by name.
Use -A / --all-namespaces to list across all namespaces.`,
		),
		Example: `  saola middleware get
  saola middleware get my-redis
  saola middleware get my-redis -o yaml
  saola middleware get -A -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|wide|yaml|json|name", "Output format: table|wide|yaml|json|name"))
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, lang.T("跨所有命名空间列出", "List across all namespaces"))

	return cmd
}

// Run executes the get/list logic.
//
// 执行 get/list 逻辑：按 name 获取单个，或列出全部。
func (o *GetOptions) Run(ctx context.Context) error {
	p, err := printer.New(o.Output)
	if err != nil {
		return err
	}

	cli := o.Client
	if cli == nil {
		var initErr error
		cli, initErr = client.New(o.Config).Get()
		if initErr != nil {
			return fmt.Errorf("create k8s client: %w", initErr)
		}
	}

	// Single-object get.
	//
	// 获取单个对象。
	if o.Name != "" {
		ns := o.Namespace
		if ns == "" {
			ns = o.Config.Namespace
		}
		if ns == "" {
			ns = "default"
		}

		mw, getErr := zeusk8s.GetMiddleware(ctx, cli, o.Name, ns)
		if getErr != nil {
			return fmt.Errorf("get middleware %s/%s: %w", ns, o.Name, getErr)
		}

		return printMiddlewares(p, []zeusv1.Middleware{*mw}, o.Output)
	}

	// List mode: respect --all-namespaces vs configured namespace.
	//
	// 列表模式：支持 --all-namespaces 或 namespace 范围。
	ns := ""
	if !o.AllNamespaces {
		ns = o.Namespace
		if ns == "" {
			ns = o.Config.Namespace
		}
		if ns == "" {
			ns = "default"
		}
	}

	items, listErr := zeusk8s.ListMiddlewares(ctx, cli, ns, sigs.MatchingLabels{})
	if listErr != nil {
		return fmt.Errorf("list middlewares: %w", listErr)
	}

	return printMiddlewares(p, items, o.Output)
}

// middlewareWideRow is the table row model for wide output (extra LABELS column).
//
// middlewareWideRow 是 wide 模式下的行结构，额外包含 LABELS 列。
type middlewareWideRow struct {
	NAME      string
	NAMESPACE string
	BASELINE  string
	STATE     string
	AGE       string
	LABELS    string
}

// printMiddlewares prints a slice of Middleware objects using the given printer.
//
// 用指定的 printer 输出 Middleware 列表，支持 table/wide/yaml/json/name 格式。
func printMiddlewares(p printer.Printer, items []zeusv1.Middleware, format string) error {
	// For structured formats, marshal the full objects.
	//
	// 结构化格式直接输出完整对象。
	if format == "yaml" || format == "json" {
		return p.Print(os.Stdout, items)
	}

	// name format: output "middleware/name" per line.
	//
	// name 格式：每行输出 "middleware/name"。
	if format == "name" {
		np, ok := p.(*printer.NamePrinter)
		if ok {
			np.ResourceType = "middleware"
		}
		return p.Print(os.Stdout, items)
	}

	// wide format: include LABELS column.
	//
	// wide 格式：额外显示 LABELS 列。
	if format == "wide" {
		rows := make([]middlewareWideRow, 0, len(items))
		for _, mw := range items {
			rows = append(rows, middlewareWideRow{
				NAME:      mw.Name,
				NAMESPACE: mw.Namespace,
				BASELINE:  mw.Spec.Baseline,
				STATE:     string(mw.Status.State),
				AGE:       formatAge(mw.CreationTimestamp.Time),
				LABELS:    formatLabelsShort(mw.Labels),
			})
		}
		return p.Print(os.Stdout, rows)
	}

	// Default: standard table rows.
	//
	// 默认：标准 table 行。
	rows := make([]middlewareRow, 0, len(items))
	for _, mw := range items {
		rows = append(rows, middlewareRow{
			NAME:      mw.Name,
			NAMESPACE: mw.Namespace,
			BASELINE:  mw.Spec.Baseline,
			STATE:     string(mw.Status.State),
			AGE:       formatAge(mw.CreationTimestamp.Time),
		})
	}
	return p.Print(os.Stdout, rows)
}

// formatLabelsShort converts a label map to a compact "k=v,k=v" string.
// Keys are sorted for deterministic output. Returns "<none>" if the map is empty.
//
// 将 label map 转换为紧凑的 "k=v,k=v" 字符串。
// key 排序以确保输出确定性；map 为空时返回 "<none>"。
func formatLabelsShort(labels map[string]string) string {
	if len(labels) == 0 {
		return "<none>"
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ",")
}

// formatAge returns a human-readable duration string since t.
//
// 返回从 t 到现在的可读时长字符串。
func formatAge(t time.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	d := time.Since(t).Round(time.Second)
	if d < 0 {
		return "<just now>"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
