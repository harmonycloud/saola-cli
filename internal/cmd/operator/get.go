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

package operator

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/cmdutil"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/printer"
	zeusv1 "github.com/opensaola/opensaola/api/v1"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOptions holds parameters for "operator get".
//
// GetOptions 保存 "operator get" 命令的参数。
type GetOptions struct {
	Config        *config.Config
	Name          string
	Namespace     string
	Output        string
	AllNamespaces bool
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// operatorRow is a display-friendly row for table output.
//
// operatorRow 是用于表格输出的显示行结构。
type operatorRow struct {
	Name      string
	Namespace string
	Baseline  string
	State     string
	Ready     string
	Runtime   string
	Age       string
}

// NewCmdGet returns the operator get command.
//
// 返回 operator get 子命令。
func NewCmdGet(cfg *config.Config) *cobra.Command {
	o := &GetOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "get [name]",
		Short: lang.T("列出或查询 MiddlewareOperator 资源", "List or get MiddlewareOperator resources"),
		Long: lang.T(
			`列出所有 MiddlewareOperator 资源，或按名称查询单个资源。使用 -o yaml/json 可输出完整资源信息。`,
			`List all MiddlewareOperator resources or get a single one by name. Use -o yaml/json for full resource output.`,
		),
		Example: `  saola operator get
  saola operator get redis-operator -n my-ns
  saola operator get -A -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|wide|yaml|json|name", "Output format: table|wide|yaml|json|name"))
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, lang.T("列出所有命名空间下的资源", "List across all namespaces"))

	return cmd
}

// Run executes the get/list logic.
//
// Run 执行 get/list 逻辑。
func (o *GetOptions) Run(ctx context.Context) error {
	p, err := printer.New(o.Output)
	if err != nil {
		return err
	}

	// Use the injected client if provided, otherwise build one from config.
	//
	// 优先使用注入的 client，否则根据 config 创建。
	cli := o.Client
	if cli == nil {
		var buildErr error
		cli, buildErr = client.New(o.Config).Get()
		if buildErr != nil {
			return fmt.Errorf("create k8s client: %w", buildErr)
		}
	}

	if o.Name != "" {
		return o.getSingle(ctx, cli, p)
	}
	return o.list(ctx, cli, p)
}

// getSingle fetches one MiddlewareOperator by name.
//
// getSingle 按名称获取单个 MiddlewareOperator。
func (o *GetOptions) getSingle(ctx context.Context, cli sigs.Client, p printer.Printer) error {
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}
	// MiddlewareOperator requires an explicit namespace to prevent accidental operations
	// across namespaces, as operators manage cluster-wide middleware types.
	//
	// MiddlewareOperator 要求显式指定 namespace，以防止跨 namespace 的误操作，因为 operator 管理的是集群级别的中间件类型。
	if ns == "" {
		return fmt.Errorf("namespace is required: specify --namespace or set SAOLA_NAMESPACE")
	}

	mo := &zeusv1.MiddlewareOperator{}
	if err := cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: ns}, mo); err != nil {
		return fmt.Errorf("get MiddlewareOperator %s/%s: %w", ns, o.Name, err)
	}

	switch o.Output {
	case "yaml", "json":
		return p.Print(os.Stdout, mo)
	case "name":
		// Output "operator/name" format.
		//
		// 输出 "operator/name" 格式。
		np, ok := p.(*printer.NamePrinter)
		if ok {
			np.ResourceType = "operator"
		}
		return p.Print(os.Stdout, mo)
	case "wide":
		// Wide: include LABELS column.
		//
		// wide 模式：添加 LABELS 列。
		rows := [][]string{
			{"NAME", "NAMESPACE", "BASELINE", "STATE", "READY", "RUNTIME", "AGE", "LABELS"},
			toWideRow(mo),
		}
		return p.Print(os.Stdout, rows)
	default:
		// Standard table output.
		//
		// 标准 table 输出。
		rows := [][]string{
			{"NAME", "NAMESPACE", "BASELINE", "STATE", "READY", "RUNTIME", "AGE"},
			toRow(mo),
		}
		return p.Print(os.Stdout, rows)
	}
}

// list fetches all MiddlewareOperators in a namespace (or all namespaces).
//
// list 获取指定命名空间（或所有命名空间）下的所有 MiddlewareOperator。
func (o *GetOptions) list(ctx context.Context, cli sigs.Client, p printer.Printer) error {
	ns := ""
	if !o.AllNamespaces {
		ns = o.Namespace
		if ns == "" {
			ns = o.Config.Namespace
		}
	}

	moList := &zeusv1.MiddlewareOperatorList{}
	listOpts := []sigs.ListOption{}
	if ns != "" {
		listOpts = append(listOpts, sigs.InNamespace(ns))
	}
	if err := cli.List(ctx, moList, listOpts...); err != nil {
		return fmt.Errorf("list MiddlewareOperators: %w", err)
	}

	switch o.Output {
	case "yaml", "json":
		return p.Print(os.Stdout, moList)
	case "name":
		// Output "operator/name" per line.
		//
		// 每行输出 "operator/name"。
		np, ok := p.(*printer.NamePrinter)
		if ok {
			np.ResourceType = "operator"
		}
		return p.Print(os.Stdout, moList)
	case "wide":
		// Wide: include LABELS column.
		//
		// wide 模式：添加 LABELS 列。
		rows := [][]string{
			{"NAME", "NAMESPACE", "BASELINE", "STATE", "READY", "RUNTIME", "AGE", "LABELS"},
		}
		for i := range moList.Items {
			rows = append(rows, toWideRow(&moList.Items[i]))
		}
		return p.Print(os.Stdout, rows)
	default:
		// Standard table output.
		//
		// 标准 table 输出。
		rows := [][]string{
			{"NAME", "NAMESPACE", "BASELINE", "STATE", "READY", "RUNTIME", "AGE"},
		}
		for i := range moList.Items {
			rows = append(rows, toRow(&moList.Items[i]))
		}
		return p.Print(os.Stdout, rows)
	}
}

// toRow converts a MiddlewareOperator to a standard table row.
//
// toRow 把 MiddlewareOperator 转换为标准表格行。
func toRow(mo *zeusv1.MiddlewareOperator) []string {
	ready := "false"
	if mo.Status.Ready {
		ready = "true"
	}
	age := "-"
	if !mo.CreationTimestamp.IsZero() {
		age = cmdutil.FormatAge(time.Since(mo.CreationTimestamp.Time))
	}
	return []string{
		mo.Name,
		mo.Namespace,
		mo.Spec.Baseline,
		string(mo.Status.State),
		ready,
		mo.Status.Runtime,
		age,
	}
}

// toWideRow converts a MiddlewareOperator to a wide table row (includes LABELS).
//
// toWideRow 把 MiddlewareOperator 转换为 wide 表格行（包含 LABELS 列）。
func toWideRow(mo *zeusv1.MiddlewareOperator) []string {
	return append(toRow(mo), formatLabelsShort(mo.Labels))
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

