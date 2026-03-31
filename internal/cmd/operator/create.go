package operator

import (
	"context"
	"fmt"
	"os"

	zeusv1 "gitea.com/middleware-management/zeus-operator/api/v1"
	"gitea.com/middleware-management/saola-cli/internal/client"
	"gitea.com/middleware-management/saola-cli/internal/config"
	"gitea.com/middleware-management/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	// sigsyaml recognises json struct tags, which is required for
	// correctly decoding Kubernetes ObjectMeta fields.
	//
	// sigsyaml 能识别 json struct tag，是正确解码 Kubernetes ObjectMeta 的必要条件。
	sigsyaml "sigs.k8s.io/yaml"
)

// CreateOptions holds parameters for "operator create".
//
// CreateOptions 保存 "operator create" 命令的参数。
type CreateOptions struct {
	Config    *config.Config
	Namespace string
	File      string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdCreate returns the "operator create" command.
//
// 返回 "operator create" 子命令。
func NewCmdCreate(cfg *config.Config) *cobra.Command {
	o := &CreateOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "create",
		Short: lang.T("从 YAML 文件创建 MiddlewareOperator 资源", "Create a MiddlewareOperator resource from a YAML file"),
		Long: lang.T(
			`从指定的 YAML 文件读取 MiddlewareOperator 清单并在集群中创建。`,
			`Read a MiddlewareOperator manifest from a YAML file and create it in the cluster.`,
		),
		Example: `  # 从清单文件创建 / Create from a manifest file
  saola operator create -f operator.yaml

  # 覆盖命名空间 / Override namespace
  saola operator create -f operator.yaml --namespace my-ns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("覆盖清单中的命名空间", "Override the namespace from the manifest"))
	cmd.Flags().StringVarP(&o.File, "file", "f", "", lang.T("YAML 清单文件路径（必填）", "Path to a MiddlewareOperator manifest YAML file (required)"))
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

// Run executes the create logic.
//
// Run 执行创建逻辑。
func (o *CreateOptions) Run(ctx context.Context) error {
	// Read and unmarshal the manifest file.
	//
	// 读取并反序列化 YAML 文件。
	raw, err := os.ReadFile(o.File)
	if err != nil {
		return fmt.Errorf("read file %q: %w", o.File, err)
	}

	mo := &zeusv1.MiddlewareOperator{}
	if err = sigsyaml.Unmarshal(raw, mo); err != nil {
		return fmt.Errorf("unmarshal MiddlewareOperator: %w", err)
	}

	// --namespace overrides whatever is in the manifest.
	//
	// --namespace 覆盖 manifest 中的 namespace。
	if o.Namespace != "" {
		mo.Namespace = o.Namespace
	} else if o.Config.Namespace != "" && mo.Namespace == "" {
		mo.Namespace = o.Config.Namespace
	}

	if mo.Namespace == "" {
		return fmt.Errorf("namespace is required: specify --namespace or set it in the manifest")
	}

	// Use the injected client if provided, otherwise build one from config.
	//
	// 优先使用注入的 client，否则根据 config 创建。
	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	// Check for duplicates before creating.
	//
	// 创建前检查是否已存在同名资源。
	existing := &zeusv1.MiddlewareOperator{}
	getErr := cli.Get(ctx, sigs.ObjectKey{Name: mo.Name, Namespace: mo.Namespace}, existing)
	if getErr == nil {
		return fmt.Errorf("MiddlewareOperator %s/%s already exists", mo.Namespace, mo.Name)
	}
	if !apierrors.IsNotFound(getErr) {
		return fmt.Errorf("check existing MiddlewareOperator: %w", getErr)
	}

	if err = cli.Create(ctx, mo); err != nil {
		return fmt.Errorf("create MiddlewareOperator: %w", err)
	}

	fmt.Fprintf(os.Stdout, "middlewareoperator/%s created\n", mo.Name)
	return nil
}
