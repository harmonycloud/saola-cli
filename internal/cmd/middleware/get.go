package middleware

import (
	"context"
	"fmt"
	"os"
	"time"

	zeusk8s "gitea.com/middleware-management/zeus-operator/pkg/k8s"
	zeusv1 "gitea.com/middleware-management/zeus-operator/api/v1"
	"gitea.com/middleware-management/saola-cli/internal/client"
	"gitea.com/middleware-management/saola-cli/internal/config"
	"gitea.com/middleware-management/saola-cli/internal/lang"
	"gitea.com/middleware-management/saola-cli/internal/printer"
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
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
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

// printMiddlewares prints a slice of Middleware objects using the given printer.
//
// 用指定的 printer 输出 Middleware 列表。
func printMiddlewares(p printer.Printer, items []zeusv1.Middleware, format string) error {
	// For non-table formats, marshal the full objects.
	//
	// 非 table 格式直接输出完整对象。
	if format == "yaml" || format == "json" {
		return p.Print(os.Stdout, items)
	}

	// Build table rows.
	//
	// 构建 table 行。
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
