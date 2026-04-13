package pkgcmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/packager"
	"gitee.com/opensaola/saola-cli/internal/waiter"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// InstallOptions holds parameters for "package install".
// InstallOptions 保存 "package install" 命令的参数。
type InstallOptions struct {
	Config *config.Config
	PkgDir string
	Name   string
	Wait   time.Duration
	DryRun bool
	// Client is the k8s client to use. If nil, a real client is built from Config.
	//
	// Client 为注入的 k8s 客户端；为 nil 时从 Config 构建真实客户端（供测试注入 fake client）。
	Client sigs.Client
}

// NewCmdInstall returns the "package install" command.
//
// 返回 package install 子命令。
func NewCmdInstall(cfg *config.Config) *cobra.Command {
	o := &InstallOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "install <pkg-dir>",
		Short: lang.T("从本地目录安装中间件包", "Install a middleware package from a local directory"),
		Long: lang.T(
			`将本地包目录打包并在 pkg-namespace 中创建 Secret。
zeus-operator 检测到 Secret 后会自动安装该包。`,
			`Pack the local package directory and create a Secret in the pkg-namespace.
zeus-operator picks up the Secret and installs the package automatically.`,
		),
		Example: `  # 从当前目录安装，名称从 metadata.yaml 自动获取 / Install from current directory, auto-name from metadata.yaml
  saola package install .

  # 指定 Secret 名称并等待最多 5 分钟 / Install with an explicit Secret name and wait up to 5 minutes
  saola package install ./my-redis --name redis-v1 --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&o.Name, "name", "", lang.T("覆盖 Secret 名称（默认：<name>-<version>）", "Override the Secret name (default: <name>-<version>)"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待安装完成（如 5m，0 表示不等待）", "Wait for installation to complete (e.g. 5m, 0 = don't wait)"))
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, lang.T("打印 Secret 清单而不实际应用", "Print the Secret manifest without applying it"))

	return cmd
}

func (o *InstallOptions) Run(ctx context.Context) error {
	// 1. Pack the directory.
	fmt.Fprintf(os.Stdout, "Packing directory %s ...\n", o.PkgDir)
	data, meta, err := packager.PackDir(o.PkgDir)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Packed %s@%s (%d bytes compressed)\n", meta.Name, meta.Version, len(data))

	// 2. Build the install Secret.
	secret := packager.BuildInstallSecret(o.Name, o.Config.PkgNamespace, meta, data)

	if o.DryRun {
		fmt.Fprintf(os.Stdout, "Dry-run: would create Secret %s/%s\n", secret.Namespace, secret.Name)
		return nil
	}

	// 3. Apply to the cluster.
	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	existing := secret.DeepCopy()
	getErr := cli.Get(ctx, sigs.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, existing)
	if getErr != nil && !apierrors.IsNotFound(getErr) {
		return fmt.Errorf("check existing Secret: %w", getErr)
	}
	if apierrors.IsNotFound(getErr) {
		if err = cli.Create(ctx, secret); err != nil {
			return fmt.Errorf("create Secret: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Secret %s/%s created\n", secret.Namespace, secret.Name)
	} else {
		return fmt.Errorf("Secret %s/%s already exists; use 'package upgrade' to update", secret.Namespace, secret.Name)
	}

	// 4. Optionally wait.
	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, "Waiting up to %s for installation to complete...\n", o.Wait)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()
		if err = waiter.WaitForInstall(waitCtx, cli, secret.Name, secret.Namespace); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Package %s installed successfully\n", secret.Name)
	}
	return nil
}
