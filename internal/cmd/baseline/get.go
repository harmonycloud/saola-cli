package baseline

import (
	"context"
	"fmt"
	"os"

	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/printer"
	"gitee.com/opensaola/opensaola/pkg/service/packages"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOptions holds parameters for "baseline get".
// GetOptions 保存 "baseline get" 命令的参数。
type GetOptions struct {
	Config  *config.Config
	Name    string
	Package string
	Kind    string
	Output  string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdGet returns the "baseline get" command.
//
// 返回 baseline get 子命令。
func NewCmdGet(cfg *config.Config) *cobra.Command {
	o := &GetOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: lang.T("从已安装包中获取指定 baseline", "Get a specific baseline from an installed package"),
		Long: lang.T(
			`从已安装的中间件包中按名称获取单个 MiddlewareBaseline 或 MiddlewareOperatorBaseline。
使用 --kind 指定 baseline 类型（middleware 或 operator）。`,
			`Fetch a single MiddlewareBaseline or MiddlewareOperatorBaseline from an installed package by name.
Use --kind to specify the baseline type (middleware or operator).`,
		),
		Example: `  saola baseline get default --package redis-v1
  saola baseline get default --package redis-v1 --kind operator -o yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&o.Package, "package", "", lang.T("要查询的包名（必填）", "Package name to query (required)"))
	cmd.Flags().StringVar(&o.Kind, "kind", "middleware", lang.T("Baseline 类型：middleware|operator", "Baseline kind: middleware|operator"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "yaml", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	_ = cmd.MarkFlagRequired("package")
	return cmd
}

func (o *GetOptions) Run(ctx context.Context) error {
	packages.SetDataNamespace(o.Config.PkgNamespace)

	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	p, err := printer.New(o.Output)
	if err != nil {
		return err
	}

	switch o.Kind {
	case "middleware":
		baseline, err := packages.GetMiddlewareBaseline(ctx, cli, o.Name, o.Package)
		if err != nil {
			return fmt.Errorf("get middleware baseline: %w", err)
		}
		return p.Print(os.Stdout, baseline)
	case "operator":
		baseline, err := packages.GetMiddlewareOperatorBaseline(ctx, cli, o.Name, o.Package)
		if err != nil {
			return fmt.Errorf("get operator baseline: %w", err)
		}
		return p.Print(os.Stdout, baseline)
	default:
		return fmt.Errorf("unknown baseline kind %q (supported: middleware, operator)", o.Kind)
	}
}
