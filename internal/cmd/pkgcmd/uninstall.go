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

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/client"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/harmonycloud/saola-cli/internal/waiter"
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
			`真实卸载中间件包。命令会先检查是否仍有 Middleware 或 MiddlewareOperator 引用该包；
检查通过后给包 Secret 添加清理 finalizer 并发起删除，由 OpenSaola 清理包资源后移除 finalizer。`,
			`Really uninstall a middleware package. The command first checks whether any Middleware or MiddlewareOperator still references the package;
after the check passes, it adds a cleanup finalizer to the package Secret and deletes it; OpenSaola removes the finalizer after cleaning package resources.`,
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

	usages, err := findPackageUsages(ctx, cli, o.Name)
	if err != nil {
		return err
	}
	if err = packageUsageError(o.Name, usages); err != nil {
		return err
	}

	if secret.GetDeletionTimestamp() != nil && !hasString(secret.Finalizers, finalizerPackageSecret) {
		return fmt.Errorf("package Secret %q is already deleting without the OpenSaola package finalizer; cleanup cannot be guaranteed", o.Name)
	}
	if secret.GetDeletionTimestamp() == nil {
		patch := sigs.MergeFrom(secret.DeepCopy())
		if !hasString(secret.Finalizers, finalizerPackageSecret) {
			secret.Finalizers = append(secret.Finalizers, finalizerPackageSecret)
		}
		if secret.Annotations != nil {
			delete(secret.Annotations, zeusv1.LabelUnInstall)
			delete(secret.Annotations, annotationUninstallError)
		}
		if err = cli.Patch(ctx, secret, patch); err != nil {
			return fmt.Errorf("patch Secret finalizer: %w", err)
		}
		if err = cli.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete Secret: %w", err)
		}
	}
	fmt.Fprintf(os.Stdout, "Uninstall deletion requested for package Secret %s/%s\n", o.Config.PkgNamespace, o.Name)

	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, "Waiting up to %s for uninstallation to complete...\n", o.Wait)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()
		if err = waiter.WaitForPackageDeleted(waitCtx, cli, o.Name, o.Config.PkgNamespace); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Package %s uninstalled successfully\n", o.Name)
	}
	return nil
}

func hasString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
