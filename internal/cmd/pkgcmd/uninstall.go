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

package pkgcmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/waiter"
	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// UninstallOptions holds parameters for "package uninstall".
// UninstallOptions 保存 "package uninstall" 命令的参数。
type UninstallOptions struct {
	Config *config.Config
	Name   string
	Wait   time.Duration
	// Client is the k8s client to use. If nil, a real client is built from Config.
	//
	// Client 为注入的 k8s 客户端；为 nil 时从 Config 构建真实客户端（供测试注入 fake client）。
	Client sigs.Client
}

// NewCmdUninstall returns the "package uninstall" command.
//
// 返回 package uninstall 子命令。
func NewCmdUninstall(cfg *config.Config) *cobra.Command {
	o := &UninstallOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "uninstall <name>",
		Short: lang.T("卸载中间件包", "Uninstall a middleware package"),
		Long: lang.T(
			`在包对应的 Secret 上添加卸载注解。
OpenSaola 检测到注解后会自动卸载该包。`,
			`Add the uninstall annotation to the package Secret.
OpenSaola will pick up the annotation and uninstall the package.`,
		),
		Example: `  saola package uninstall redis-v1
  saola package uninstall redis-v1 --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待卸载完成（0 表示不等待）", "Wait for uninstallation to complete (0 = don't wait)"))
	return cmd
}

func (o *UninstallOptions) Run(ctx context.Context) error {
	var err error
	cli := o.Client
	if cli == nil {
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	secret := &corev1.Secret{}
	if err = cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: o.Config.PkgNamespace}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("package Secret %q not found in namespace %q", o.Name, o.Config.PkgNamespace)
		}
		return fmt.Errorf("get Secret: %w", err)
	}

	// Patch: add the uninstall annotation.
	patch := sigs.MergeFrom(secret.DeepCopy())
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	secret.Annotations[zeusv1.LabelUnInstall] = "true"
	if err = cli.Patch(ctx, secret, patch); err != nil {
		return fmt.Errorf("patch Secret: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Uninstall annotation added to Secret %s/%s\n", o.Config.PkgNamespace, o.Name)

	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, "Waiting up to %s for uninstallation to complete...\n", o.Wait)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()
		if err = waiter.WaitForUninstall(waitCtx, cli, o.Name, o.Config.PkgNamespace); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Package %s uninstalled successfully\n", o.Name)
	}
	return nil
}
