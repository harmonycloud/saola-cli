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
	zeusk8s "gitee.com/opensaola/saola-cli/internal/k8s"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

const upgradePollInterval = 2 * time.Second

// UpgradeOptions holds parameters for "operator upgrade".
//
// UpgradeOptions 保存 "operator upgrade" 命令的参数。
type UpgradeOptions struct {
	Config    *config.Config
	Name      string
	Namespace string
	ToVersion string
	Baseline  string
	Wait      time.Duration
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdUpgrade returns the "operator upgrade" command.
//
// 返回 "operator upgrade" 子命令。
func NewCmdUpgrade(cfg *config.Config) *cobra.Command {
	o := &UpgradeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "operator <name>",
		Aliases: []string{"op"},
		Short:   lang.T("升级 MiddlewareOperator 实例到指定版本", "Upgrade a MiddlewareOperator instance to a specified version"),
		Long: lang.T(
			`通过设置 middleware.cn/update 和 middleware.cn/baseline 注解触发 Controller 执行升级。`+
				`Controller 处理时会保留 Globe 和 PreActions，清空其余 Spec，切换到新 Baseline。`,
			`Trigger an upgrade by setting the middleware.cn/update and middleware.cn/baseline annotations.`+
				` The controller retains Globe and PreActions, clears the rest of Spec, and switches to the new Baseline.`,
		),
		Example: `  # 升级到指定版本 / Upgrade to a specific version
  saola upgrade operator redis-op --to-version 1.2.0 -n my-ns

  # 同时切换 baseline / Also switch baseline
  saola upgrade operator redis-op --to-version 1.2.0 --baseline redis-ha -n my-ns

  # 升级后等待完成 / Wait for upgrade to complete
  saola upgrade op redis-op --to-version 1.2.0 -n my-ns --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间（未全局设置时必填）", "Target namespace (required if not set globally)"))
	cmd.Flags().StringVar(&o.ToVersion, "to-version", "", lang.T("升级目标版本号（必填）", "Target version to upgrade to (required)"))
	cmd.Flags().StringVar(&o.Baseline, "baseline", "", lang.T("升级目标 Baseline 名称（可选，默认保留当前 Baseline）", "Target Baseline name (optional, defaults to current Baseline)"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待升级完成的超时时间（如 5m；0 表示不等待）", "Wait for upgrade to complete (e.g. 5m; 0 = don't wait)"))
	_ = cmd.MarkFlagRequired("to-version")

	return cmd
}

// Run executes the upgrade logic.
//
// Run 执行升级逻辑。
func (o *UpgradeOptions) Run(ctx context.Context) error {
	// Resolve namespace: flag > global config > error.
	//
	// 解析命名空间：flag > 全局配置 > 报错。
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

	// Fetch the MiddlewareOperator instance.
	//
	// 获取 MiddlewareOperator 实例。
	mo, err := zeusk8s.GetMiddlewareOperator(ctx, cli, o.Name, ns)
	if err != nil {
		return fmt.Errorf("get MiddlewareOperator %s/%s: %w", ns, o.Name, err)
	}

	// Determine the target baseline: use flag value or fall back to current Spec.Baseline.
	//
	// 确定目标 Baseline：优先用 flag 值，否则沿用当前 Spec.Baseline。
	baseline := o.Baseline
	if baseline == "" {
		baseline = mo.Spec.Baseline
	}
	if baseline == "" {
		return fmt.Errorf("baseline is required: specify --baseline or ensure spec.baseline is set")
	}

	// Guard against concurrent upgrades: reject if middleware.cn/update annotation already exists.
	//
	// 防止并发升级：若 middleware.cn/update 注解已存在，则拒绝本次操作。
	if mo.Annotations != nil {
		if _, ok := mo.Annotations[zeusv1.LabelUpdate]; ok {
			return fmt.Errorf(
				"middlewareoperator/%s is already upgrading (annotation %s is set); wait for it to complete first",
				o.Name, zeusv1.LabelUpdate,
			)
		}
	}

	// Set upgrade annotations to trigger the controller.
	//
	// 设置升级注解，触发 Controller 处理。
	if mo.Annotations == nil {
		mo.Annotations = make(map[string]string)
	}
	mo.Annotations[zeusv1.LabelUpdate] = o.ToVersion
	mo.Annotations[zeusv1.LabelBaseline] = baseline

	if err = cli.Update(ctx, mo); err != nil {
		return fmt.Errorf("update MiddlewareOperator %s/%s: %w", ns, o.Name, err)
	}

	fmt.Fprintf(os.Stdout, "middlewareoperator/%s upgrade triggered\n", o.Name)

	// Optionally wait for the upgrade to complete.
	//
	// 可选：等待升级完成。
	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, "Waiting up to %s for middlewareoperator/%s to become Available...\n", o.Wait, o.Name)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()

		if err = waitForUpgrade(waitCtx, cli, o.Name, ns); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "middlewareoperator/%s upgraded successfully\n", o.Name)
	}

	return nil
}

// waitForUpgrade polls until the MiddlewareOperator reaches StateAvailable (upgrade complete),
// or the context deadline is exceeded.
//
// waitForUpgrade 轮询直到 MiddlewareOperator 状态变为 Available（升级完成），或 context 超时。
func waitForUpgrade(ctx context.Context, cli sigs.Client, name, namespace string) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for middlewareoperator/%s to upgrade: %w", name, ctx.Err())
		case <-time.After(upgradePollInterval):
		}

		mo := &zeusv1.MiddlewareOperator{}
		if err := cli.Get(ctx, sigs.ObjectKey{Name: name, Namespace: namespace}, mo); err != nil {
			return fmt.Errorf("poll middlewareoperator/%s: %w", name, err)
		}

		// Upgrade is still in progress while the annotation is present or state is Updating.
		//
		// 注解仍存在或状态为 Updating 时，升级尚未完成。
		if mo.Annotations != nil {
			if _, ok := mo.Annotations[zeusv1.LabelUpdate]; ok {
				continue
			}
		}

		switch mo.Status.State {
		case zeusv1.StateAvailable:
			return nil
		case zeusv1.StateUnavailable:
			reason := mo.Status.Reason
			if reason == "" {
				reason = "unknown error"
			}
			return fmt.Errorf("middlewareoperator/%s upgrade failed: %s", name, reason)
		}
		// StateUpdating or empty — keep polling.
		//
		// 状态为 Updating 或空时继续轮询。
	}
}
