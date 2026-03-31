package operator

import (
	"context"
	"fmt"
	"os"
	"time"

	zeusv1 "gitea.com/middleware-management/zeus-operator/api/v1"
	"gitea.com/middleware-management/saola-cli/internal/client"
	"gitea.com/middleware-management/saola-cli/internal/config"
	"gitea.com/middleware-management/saola-cli/internal/lang"
	"gitea.com/middleware-management/saola-cli/internal/printer"
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
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
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
	if ns == "" {
		return fmt.Errorf("namespace is required: specify --namespace or set SAOLA_NAMESPACE")
	}

	mo := &zeusv1.MiddlewareOperator{}
	if err := cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: ns}, mo); err != nil {
		return fmt.Errorf("get MiddlewareOperator %s/%s: %w", ns, o.Name, err)
	}

	if o.Output == "table" || o.Output == "" {
		rows := [][]string{
			{"NAME", "NAMESPACE", "BASELINE", "STATE", "READY", "RUNTIME", "AGE"},
			toRow(mo),
		}
		return p.Print(os.Stdout, rows)
	}
	return p.Print(os.Stdout, mo)
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

	if o.Output == "table" || o.Output == "" {
		rows := [][]string{
			{"NAME", "NAMESPACE", "BASELINE", "STATE", "READY", "RUNTIME", "AGE"},
		}
		for i := range moList.Items {
			rows = append(rows, toRow(&moList.Items[i]))
		}
		return p.Print(os.Stdout, rows)
	}
	return p.Print(os.Stdout, moList)
}

// toRow converts a MiddlewareOperator to a table row.
//
// toRow 把 MiddlewareOperator 转换为表格行。
func toRow(mo *zeusv1.MiddlewareOperator) []string {
	ready := "false"
	if mo.Status.Ready {
		ready = "true"
	}
	age := "-"
	if !mo.CreationTimestamp.IsZero() {
		age = formatAge(time.Since(mo.CreationTimestamp.Time))
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

// formatAge formats a duration into a human-readable age string.
//
// formatAge 将 duration 格式化为人类可读的 age 字符串。
func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
