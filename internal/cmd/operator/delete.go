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

package operator

import (
	"context"
	"fmt"
	"os"
	"time"

	zeusv1 "github.com/OpenSaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

const deletePollInterval = 2 * time.Second

// DeleteOptions holds parameters for "operator delete".
//
// DeleteOptions 保存 "operator delete" 命令的参数。
type DeleteOptions struct {
	Config    *config.Config
	Name      string
	Namespace string
	Wait      time.Duration
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdDelete returns the "operator delete" command.
//
// 返回 "operator delete" 子命令。
func NewCmdDelete(cfg *config.Config) *cobra.Command {
	o := &DeleteOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: lang.T("删除 MiddlewareOperator 资源", "Delete a MiddlewareOperator resource"),
		Long: lang.T(
			`按名称删除 MiddlewareOperator。使用 --wait 可阻塞等待资源完全移除（Finalizer 清理可能需要一定时间）。`,
			`Delete a MiddlewareOperator by name. Use --wait to block until the resource is fully removed (Finalizer cleanup may take time).`,
		),
		Example: `  saola operator delete redis-operator --namespace my-ns
  saola operator delete redis-operator -n my-ns --wait 2m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间（未全局设置时必填）", "Target namespace (required if not set globally)"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待删除完成的超时时间（如 2m；0 表示不等待）", "Wait for the resource to be fully deleted (e.g. 2m; 0 = don't wait)"))

	return cmd
}

// Run executes the delete logic.
//
// Run 执行删除逻辑。
func (o *DeleteOptions) Run(ctx context.Context) error {
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}
	if ns == "" {
		return fmt.Errorf("namespace is required: specify --namespace or set SAOLA_NAMESPACE")
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

	// Fetch the object first so we can pass it to Delete.
	//
	// 先获取对象，再传给 Delete。
	mo := &zeusv1.MiddlewareOperator{}
	if err := cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: ns}, mo); err != nil {
		if apierrors.IsNotFound(err) {
			fmt.Fprintf(os.Stdout, "middlewareoperator/%s not found (already deleted)\n", o.Name)
			return nil
		}
		return fmt.Errorf("get MiddlewareOperator: %w", err)
	}

	if err := cli.Delete(ctx, mo); err != nil {
		if apierrors.IsNotFound(err) {
			fmt.Fprintf(os.Stdout, "middlewareoperator/%s not found (already deleted)\n", o.Name)
			return nil
		}
		return fmt.Errorf("delete MiddlewareOperator: %w", err)
	}

	fmt.Fprintf(os.Stdout, "middlewareoperator/%s deleted\n", o.Name)

	// BUG-5: --wait polls until the object no longer exists.
	// Finalizer cleanup by the controller can take considerable time.
	//
	// BUG-5: --wait 轮询直到对象不存在，Finalizer 清理需要时间。
	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, "Waiting up to %s for middlewareoperator/%s to be fully removed...\n", o.Wait, o.Name)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()

		if err := waitForDeletion(waitCtx, cli, o.Name, ns); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "middlewareoperator/%s fully removed\n", o.Name)
	}

	return nil
}

// waitForDeletion polls until the MiddlewareOperator no longer exists or the context is cancelled.
//
// waitForDeletion 轮询直到 MiddlewareOperator 不再存在，或 context 被取消。
func waitForDeletion(ctx context.Context, cli sigs.Client, name, namespace string) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for middlewareoperator/%s to be deleted: %w", name, ctx.Err())
		case <-time.After(deletePollInterval):
		}

		check := &zeusv1.MiddlewareOperator{}
		err := cli.Get(ctx, sigs.ObjectKey{Name: name, Namespace: namespace}, check)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("poll middlewareoperator/%s: %w", name, err)
		}
	}
}
