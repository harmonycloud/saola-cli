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

package middleware

import (
	"context"
	"fmt"
	"os"
	"time"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/client"
	"github.com/harmonycloud/saola-cli/internal/config"
	zeusk8s "github.com/harmonycloud/saola-cli/internal/k8s"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteOptions holds parameters for "middleware delete".
//
// DeleteOptions 保存 middleware delete 子命令的所有参数。
type DeleteOptions struct {
	Config    *config.Config
	Name      string
	Namespace string
	Wait      time.Duration
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；为 nil 时使用 client.New(cfg)。
	Client sigs.Client
}

// NewCmdDelete returns the middleware delete command.
//
// 返回 middleware delete 子命令。
func NewCmdDelete(cfg *config.Config) *cobra.Command {
	o := &DeleteOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: lang.T("删除 Middleware 资源", "Delete a Middleware resource"),
		Long: lang.T(
			`按名称删除 Middleware 资源，资源不存在时静默跳过。`,
			`Delete a Middleware resource by name. Tolerates NotFound.`,
		),
		Example: `  saola middleware delete my-redis
  saola middleware delete my-redis -n production
  saola middleware delete my-redis --wait 2m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待删除完成的超时（如 2m，0 不等待）", "Poll timeout (e.g. 2m, 0 = no wait)"))

	return cmd
}

// Run executes the delete logic.
//
// 执行删除逻辑：获取对象、调用删除、可选轮询直到对象消失。
func (o *DeleteOptions) Run(ctx context.Context) error {
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}
	// Middleware resources default to the "default" namespace when no namespace is specified,
	// matching kubectl behavior for namespaced resources.
	//
	// Middleware 资源在未指定 namespace 时默认使用 "default"，与 kubectl 对 namespaced 资源的行为一致。
	if ns == "" {
		ns = "default"
	}

	cli := o.Client
	if cli == nil {
		var initErr error
		cli, initErr = client.New(o.Config).Get()
		if initErr != nil {
			return fmt.Errorf("create k8s client: %w", initErr)
		}
	}

	// Fetch the object first so we have the full resource for deletion.
	//
	// 先 Get 拿到完整对象，再执行删除。
	mw, err := zeusk8s.GetMiddleware(ctx, cli, o.Name, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			fmt.Fprintf(os.Stdout, "middleware/%s not found (already deleted)\n", o.Name)
			return nil
		}
		return fmt.Errorf("get middleware %s/%s: %w", ns, o.Name, err)
	}

	if err = zeusk8s.DeleteMiddleware(ctx, cli, mw); err != nil {
		if apierrors.IsNotFound(err) {
			fmt.Fprintf(os.Stdout, "middleware/%s not found (already deleted)\n", o.Name)
			return nil
		}
		return fmt.Errorf("delete middleware %s/%s: %w", ns, o.Name, err)
	}

	fmt.Fprintf(os.Stdout, "middleware/%s deleted\n", o.Name)

	// Optionally wait until the object is fully gone.
	//
	// 可选：轮询直到对象不再存在。
	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, "Waiting up to %s for middleware/%s to be deleted...\n", o.Wait, o.Name)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()

		err = wait.PollUntilContextCancel(waitCtx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
			probe := &zeusv1.Middleware{
				ObjectMeta: metav1.ObjectMeta{
					Name:      o.Name,
					Namespace: ns,
				},
			}
			getErr := cli.Get(ctx, sigs.ObjectKeyFromObject(probe), probe)
			if apierrors.IsNotFound(getErr) {
				return true, nil
			}
			if getErr != nil {
				return false, getErr
			}
			return false, nil
		})
		if err != nil {
			return fmt.Errorf("waiting for deletion: %w", err)
		}
		fmt.Fprintf(os.Stdout, "middleware/%s fully deleted\n", o.Name)
	}

	return nil
}
