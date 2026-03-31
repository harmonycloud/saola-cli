package middleware

import (
	"context"
	"fmt"
	"os"

	zeusv1 "gitea.com/middleware-management/zeus-operator/api/v1"
	zeusk8s "gitea.com/middleware-management/zeus-operator/pkg/k8s"
	"gitea.com/middleware-management/saola-cli/internal/client"
	"gitea.com/middleware-management/saola-cli/internal/config"
	"gitea.com/middleware-management/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	sigsyaml "sigs.k8s.io/yaml"
)

// CreateOptions holds parameters for "middleware create".
//
// CreateOptions 保存 middleware create 子命令的所有参数。
type CreateOptions struct {
	Config    *config.Config
	Namespace string
	File      string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；为 nil 时使用 client.New(cfg)。
	Client sigs.Client
}

// NewCmdCreate returns the middleware create command.
//
// 返回 middleware create 子命令。
func NewCmdCreate(cfg *config.Config) *cobra.Command {
	o := &CreateOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "create",
		Short: lang.T("从 YAML 文件创建 Middleware 资源", "Create a Middleware resource from a YAML file"),
		Long: lang.T(
			`从指定的 YAML 文件读取 Middleware 清单并在集群中创建。`,
			`Read a Middleware manifest from a YAML file and create it in the cluster.`,
		),
		Example: `  # 从清单文件创建 / Create from a manifest file
  saola middleware create -f middleware.yaml

  # 覆盖命名空间 / Override namespace
  saola middleware create -f middleware.yaml -n production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.File, "file", "f", "", lang.T("YAML 清单文件路径（必填）", "Path to a Middleware manifest YAML file (required)"))
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("覆盖清单中的 namespace", "Override namespace from the manifest"))
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

// Run executes the create logic.
//
// 执行 create 逻辑：读取 YAML、反序列化、可选覆盖 namespace，最后调用 k8s 创建。
func (o *CreateOptions) Run(ctx context.Context) error {
	// 1. Read the manifest file.
	//
	// 读取 YAML 文件内容。
	data, err := os.ReadFile(o.File)
	if err != nil {
		return fmt.Errorf("read file %s: %w", o.File, err)
	}

	// 2. Deserialise into Middleware using sigs.k8s.io/yaml so that json struct tags
	// (e.g. metadata.name) are respected, matching standard Kubernetes manifest format.
	//
	// 使用 sigs.k8s.io/yaml 反序列化，该库先转 JSON 再解析，
	// 能正确识别 json struct tag（如 metadata.name）。
	var mw zeusv1.Middleware
	if err = sigsyaml.Unmarshal(data, &mw); err != nil {
		return fmt.Errorf("parse middleware manifest: %w", err)
	}

	// 3. Override namespace when --namespace is explicitly provided.
	//
	// 若指定了 --namespace，则覆盖 manifest 中的 namespace。
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}
	if ns != "" {
		mw.Namespace = ns
	}
	if mw.Namespace == "" {
		mw.Namespace = "default"
	}

	// 4. Create the resource via the k8s helper.
	//
	// 调用 k8s helper 创建资源。
	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	if err = zeusk8s.CreateMidddleware(ctx, cli, &mw); err != nil {
		return fmt.Errorf("create middleware %s/%s: %w", mw.Namespace, mw.Name, err)
	}

	fmt.Fprintf(os.Stdout, "middleware/%s created\n", mw.Name)
	return nil
}
