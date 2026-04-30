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
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// UpgradeOptions holds parameters for "middleware upgrade".
//
// UpgradeOptions 保存 middleware upgrade 子命令的所有参数。
type UpgradeOptions struct {
	Config    *config.Config
	Name      string        // positional arg / 位置参数
	Namespace string        // --namespace / -n
	ToVersion string        // --to-version（required / 必填）
	Baseline  string        // --baseline（optional / 可选，默认保持当前 baseline）
	Wait      time.Duration // --wait（optional / 可选）
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；为 nil 时使用 client.New(cfg)。
	Client sigs.Client
}

// NewCmdUpgrade returns the middleware upgrade command.
//
// 返回 middleware upgrade 子命令。
func NewCmdUpgrade(cfg *config.Config) *cobra.Command {
	o := &UpgradeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "middleware <name>",
		Aliases: []string{"mw"},
		Short:   lang.T("升级 Middleware 实例到指定版本", "Upgrade a Middleware instance to the specified version"),
		Long: lang.T(
			`通过在 Middleware CR 上设置 annotation 触发 OpenSaola 执行版本升级。

controller 检测到 annotation 后将执行 ReplacePackage()：
  查找目标版本包 → 切换 Spec.Baseline → 更新 Labels → 删除 annotation → State: Updating → Available`,
			`Trigger a version upgrade by patching annotations on the Middleware CR.

The controller detects the annotations and runs ReplacePackage():
  find target package → switch Spec.Baseline → update Labels → remove annotation → State: Updating → Available`,
		),
		Example: `  # 升级到指定版本 / Upgrade to a version
  saola upgrade middleware my-redis --to-version 7.2.1

  # 同时切换 baseline / Also switch baseline
  saola upgrade middleware my-redis --to-version 7.2.1 --baseline redis-standalone-v2

  # 升级并等待完成 / Upgrade and wait for completion
  saola upgrade middleware my-redis --to-version 7.2.1 --wait 5m

  # 使用别名 / Using alias
  saola upgrade mw my-redis --to-version 7.2.1 -n production`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().StringVar(&o.ToVersion, "to-version", "", lang.T("目标版本号（必填）", "Target version (required)"))
	cmd.Flags().StringVar(&o.Baseline, "baseline", "", lang.T("目标 baseline 名称（可选，默认保持当前值）", "Target baseline name (optional, defaults to current value)"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待升级完成的超时（如 5m，0 不等待）", "Poll timeout for upgrade completion (e.g. 5m, 0 = no wait)"))
	_ = cmd.MarkFlagRequired("to-version")

	return cmd
}

// Run executes the upgrade logic.
//
// 执行升级逻辑：获取对象 → 校验状态 → 设置 annotation → Update → 可选轮询等待。
func (o *UpgradeOptions) Run(ctx context.Context) error {
	// 1. Resolve namespace: flag → config → "default".
	//
	// 解析命名空间优先级：--namespace → Config.Namespace → "default"。
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

	// 2. Initialize k8s client.
	//
	// 初始化 k8s client。
	cli := o.Client
	if cli == nil {
		var initErr error
		cli, initErr = client.New(o.Config).Get()
		if initErr != nil {
			return fmt.Errorf("create k8s client: %w", initErr)
		}
	}

	// 3. Fetch the current Middleware CR.
	//
	// 获取当前 Middleware CR。
	mw, err := zeusk8s.GetMiddleware(ctx, cli, o.Name, ns)
	if err != nil {
		return fmt.Errorf("get middleware %s/%s: %w", ns, o.Name, err)
	}

	// 4. If --baseline is not set, keep the current spec.baseline unchanged.
	//
	// 若未指定 --baseline，则沿用当前 spec.baseline。
	baseline := o.Baseline
	if baseline == "" {
		baseline = mw.Spec.Baseline
	}

	// 5. Guard: reject if an upgrade is already in progress on this resource.
	//
	// 防御检查：若 annotation 已存在，说明上一次升级尚未完成，直接报错。
	if mw.Annotations != nil {
		if existing, ok := mw.Annotations[zeusv1.LabelUpdate]; ok {
			return fmt.Errorf(
				"upgrade already in progress for middleware %s/%s (pending version: %s); "+
					"wait for it to complete or remove the annotation manually",
				ns, o.Name, existing,
			)
		}
	}

	// 6. Patch the two trigger annotations.
	//
	// 设置两个触发升级的 annotation。
	if mw.Annotations == nil {
		mw.Annotations = map[string]string{}
	}
	mw.Annotations[zeusv1.LabelUpdate] = o.ToVersion
	mw.Annotations[zeusv1.LabelBaseline] = baseline

	// 7. Push the update to the API server.
	//
	// 将变更推送到 API server。
	if err = cli.Update(ctx, mw); err != nil {
		return fmt.Errorf("update middleware %s/%s: %w", ns, o.Name, err)
	}

	fmt.Fprintf(os.Stdout, "middleware/%s upgrade triggered (-> version %s, baseline %s)\n",
		o.Name, o.ToVersion, baseline)

	// 8. Optionally poll until the upgrade is complete.
	//
	// 可选：轮询等待升级完成。
	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, lang.T(
			"等待最多 %s，直到 middleware/%s 升级完成...\n",
			"Waiting up to %s for middleware/%s upgrade to complete...\n",
		), o.Wait, o.Name)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()

		if err = waitForUpgrade(waitCtx, cli, o.Name, ns); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "middleware/%s upgrade completed\n", o.Name)
	}

	return nil
}

// waitForUpgrade polls every 3 seconds until:
//   - the middleware.cn/update annotation is gone, AND
//   - status.state == Available
//
// It returns an error when ctx is canceled/timed out, embedding last-known state/reason.
//
// waitForUpgrade 每 3 秒轮询一次，直到以下条件同时满足：
//   - middleware.cn/update annotation 不再存在
//   - status.state == Available
//
// ctx 超时或取消时返回包含当前 state 和 reason 的错误。
func waitForUpgrade(ctx context.Context, cli sigs.Client, name, ns string) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mw, err := zeusk8s.GetMiddleware(ctx, cli, name, ns)
			if err != nil {
				// Transient API error; keep retrying until deadline.
				//
				// 临时 API 错误，继续重试直到超时。
				continue
			}
			_, hasUpdate := mw.Annotations[zeusv1.LabelUpdate]
			if !hasUpdate && mw.Status.State == zeusv1.StateAvailable {
				return nil
			}

		case <-ctx.Done():
			// Use a short background context for the final state fetch.
			//
			// 超时：用后台 context 做最终状态查询以获得有用的错误信息。
			fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			mw, fetchErr := zeusk8s.GetMiddleware(fetchCtx, cli, name, ns)
			if fetchErr != nil {
				return fmt.Errorf("timed out waiting for upgrade of middleware %s/%s", ns, name)
			}
			return fmt.Errorf(
				"timed out waiting for upgrade of middleware %s/%s: state=%s, reason=%s",
				ns, name, mw.Status.State, mw.Status.Reason,
			)
		}
	}
}
