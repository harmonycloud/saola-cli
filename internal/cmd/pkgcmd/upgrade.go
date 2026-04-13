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

// UpgradeOptions holds parameters for "package upgrade".
//
// UpgradeOptions 保存 "package upgrade" 命令的参数。
type UpgradeOptions struct {
	Config *config.Config
	PkgDir string
	Name   string
	Wait   time.Duration
	// Client is the k8s client to use. If nil, a real client is built from Config.
	//
	// Client 为注入的 k8s 客户端；为 nil 时从 Config 构建真实客户端（供测试注入 fake client）。
	Client sigs.Client
}

// NewCmdUpgrade returns the "package upgrade" command.
//
// 返回 "package upgrade" 命令。
func NewCmdUpgrade(cfg *config.Config) *cobra.Command {
	o := &UpgradeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "upgrade <pkg-dir>",
		Short: lang.T("升级已有的中间件包", "Upgrade an existing middleware package"),
		Long: lang.T(
			`用本地目录的新内容替换包 Secret，先删除旧 Secret 再以更新后的数据重新创建。`,
			`Replace the package Secret with new content from the local directory. The existing Secret is deleted and re-created with updated data.`,
		),
		Example: `  saola package upgrade ./my-redis
  saola package upgrade ./my-redis --name redis-custom
  saola package upgrade ./my-redis --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&o.Name, "name", "", lang.T("覆盖 Secret 名称（默认：<name>-<version>）", "Override the Secret name (default: <name>-<version>)"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("升级后等待安装完成的超时时间（如 5m，0 表示不等待）", "Wait for installation to complete after upgrade (e.g. 5m, 0 = don't wait)"))
	return cmd
}

// Run executes the upgrade logic: pack, delete old Secret, create new Secret, optionally wait.
//
// 执行升级逻辑：打包、删除旧 Secret、创建新 Secret，可选等待安装完成。
func (o *UpgradeOptions) Run(ctx context.Context) error {
	fmt.Fprintf(os.Stdout, "Packing directory %s ...\n", o.PkgDir)
	data, meta, err := packager.PackDir(o.PkgDir)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}

	secret := packager.BuildInstallSecret(o.Name, o.Config.PkgNamespace, meta, data)

	cli := o.Client
	if cli == nil {
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	// Delete the existing Secret if present (Immutable Secrets cannot be updated in-place).
	//
	// 如果存在旧 Secret 则先删除（不可变 Secret 无法原地更新）。
	existing := secret.DeepCopy()
	getErr := cli.Get(ctx, sigs.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, existing)
	if getErr != nil && !apierrors.IsNotFound(getErr) {
		return fmt.Errorf("check existing Secret: %w", getErr)
	}
	if getErr == nil {
		if err = cli.Delete(ctx, existing); err != nil {
			return fmt.Errorf("delete existing Secret: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Deleted existing Secret %s/%s\n", secret.Namespace, secret.Name)
	}

	if err = cli.Create(ctx, secret); err != nil {
		return fmt.Errorf("create Secret: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Secret %s/%s created (upgraded to %s@%s)\n", secret.Namespace, secret.Name, meta.Name, meta.Version)

	// Optionally wait for the operator to complete installation.
	//
	// 可选：等待 operator 完成安装。
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
