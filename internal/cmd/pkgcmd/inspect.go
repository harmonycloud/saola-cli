package pkgcmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"gitea.com/middleware-management/saola-cli/internal/client"
	"gitea.com/middleware-management/saola-cli/internal/config"
	"gitea.com/middleware-management/saola-cli/internal/lang"
	"gitea.com/middleware-management/saola-cli/internal/printer"
	"gitea.com/middleware-management/zeus-operator/pkg/service/packages"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// InspectOptions holds parameters for "package inspect".
// InspectOptions 保存 "package inspect" 命令的参数。
type InspectOptions struct {
	Config *config.Config
	Name   string
	Output string
	// Client is the k8s client to use. If nil, a real client is built from Config.
	//
	// Client 为注入的 k8s 客户端；为 nil 时从 Config 构建真实客户端（供测试注入 fake client）。
	Client sigs.Client
}

// NewCmdInspect returns the "package inspect" command.
//
// 返回 package inspect 子命令。
func NewCmdInspect(cfg *config.Config) *cobra.Command {
	o := &InspectOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "inspect <name>",
		Short: lang.T("查看已安装包的内容", "Inspect the contents of an installed package"),
		Long: lang.T(
			`从 pkg-namespace 中读取指定包的 Secret，解压 TAR 归档并展示包内文件列表及元数据。`,
			`Read the package Secret from the pkg-namespace, decompress the TAR archive, and display the file listing and metadata.`,
		),
		Example: `  saola package inspect redis-v1
  saola package inspect redis-v1 -o yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	return cmd
}

func (o *InspectOptions) Run(ctx context.Context) error {
	packages.SetDataNamespace(o.Config.PkgNamespace)

	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	pkg, err := packages.Get(ctx, cli, o.Name)
	if err != nil {
		return fmt.Errorf("get package: %w", err)
	}

	p, err := printer.New(o.Output)
	if err != nil {
		return err
	}

	switch o.Output {
	case "yaml", "json":
		return p.Print(os.Stdout, pkg)
	default:
		// Print metadata and file list in table format.
		fmt.Fprintf(os.Stdout, "Name:      %s\n", pkg.Name)
		fmt.Fprintf(os.Stdout, "Component: %s\n", pkg.Component)
		fmt.Fprintf(os.Stdout, "Enabled:   %v\n", pkg.Enabled)
		fmt.Fprintf(os.Stdout, "Created:   %s\n", pkg.Created)
		if pkg.Metadata != nil {
			fmt.Fprintf(os.Stdout, "Version:   %s\n", pkg.Metadata.Version)
			fmt.Fprintf(os.Stdout, "Type:      %s\n", pkg.Metadata.Type)
			fmt.Fprintf(os.Stdout, "Owner:     %s\n", pkg.Metadata.Owner)
		}
		fmt.Fprintln(os.Stdout, "\nFiles:")
		names := make([]string, 0, len(pkg.Files))
		for k := range pkg.Files {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(os.Stdout, "  %s (%d bytes)\n", name, len(pkg.Files[name]))
		}
	}
	return nil
}
