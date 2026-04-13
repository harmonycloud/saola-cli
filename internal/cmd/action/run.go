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

package action

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	zeusv1 "gitee.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// RunOptions holds parameters for "action run".
// RunOptions 保存 "action run" 命令的参数。
type RunOptions struct {
	Config     *config.Config
	Namespace  string
	Middleware string
	Baseline   string
	Params     []string
	Wait       time.Duration
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdRun returns the "action run" command.
//
// 返回 action run 子命令。
func NewCmdRun(cfg *config.Config) *cobra.Command {
	o := &RunOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "run",
		Short: lang.T("触发一个 MiddlewareAction", "Trigger a MiddlewareAction"),
		Long: lang.T(
			`创建一个 MiddlewareAction CR，对指定 Middleware 实例触发一次性运维操作。
Action 名称自动生成为 <baseline>-<unix时间戳> 以避免冲突。`,
			`Create a MiddlewareAction CR to trigger a one-off action against a Middleware instance.
The action name is auto-generated as <baseline>-<unix-timestamp> to avoid conflicts.`,
		),
		Example: `  # 执行备份操作 / Run a backup action
  saola action run --middleware my-redis --baseline redis-backup

  # 带参数执行并等待最多 5 分钟 / Run with extra parameters and wait up to 5 minutes
  saola action run --middleware my-redis --baseline redis-restore --params src=backup-001 --wait 5m`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间（默认使用配置中的命名空间）", "Target namespace (defaults to config namespace)"))
	cmd.Flags().StringVar(&o.Middleware, "middleware", "", lang.T("关联的 Middleware 实例名（必填）", "Name of the Middleware instance to run the action against (required)"))
	cmd.Flags().StringVar(&o.Baseline, "baseline", "", lang.T("MiddlewareActionBaseline 名称（必填）", "Name of the MiddlewareActionBaseline to use (required)"))
	cmd.Flags().StringArrayVar(&o.Params, "params", nil, lang.T("key=value 格式的操作参数，可逗号分隔或重复使用", "Action parameters in key=value format, comma-separated or repeatable"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待操作完成（如 5m，0 表示不等待）", "Wait for the action to complete (e.g. 5m, 0 = don't wait)"))

	_ = cmd.MarkFlagRequired("middleware")
	_ = cmd.MarkFlagRequired("baseline")

	return cmd
}

// Run executes the action run logic.
// Run 执行 action run 的核心逻辑。
func (o *RunOptions) Run(ctx context.Context) error {
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}

	// Build the action name: <baseline>-<unix-timestamp>.
	//
	// 自动生成 action 名称：<baseline>-<unix时间戳>，避免重名。
	name := fmt.Sprintf("%s-%d", o.Baseline, time.Now().Unix())

	// Parse params into a JSON map for spec.necessary.
	//
	// 将 --params key=value 解析为 JSON 存入 spec.necessary。
	necessary, err := parseParams(o.Params)
	if err != nil {
		return fmt.Errorf("parse params: %w", err)
	}

	action := &zeusv1.MiddlewareAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: zeusv1.MiddlewareActionSpec{
			MiddlewareName: o.Middleware,
			Baseline:       o.Baseline,
			Necessary:      necessary,
		},
	}

	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	if err = cli.Create(ctx, action); err != nil {
		return fmt.Errorf("create MiddlewareAction: %w", err)
	}
	fmt.Fprintf(os.Stdout, "MiddlewareAction %s/%s created\n", ns, name)

	if o.Wait > 0 {
		fmt.Fprintf(os.Stdout, "Waiting up to %s for action to complete...\n", o.Wait)
		waitCtx, cancel := context.WithTimeout(ctx, o.Wait)
		defer cancel()
		if err = waitForAction(waitCtx, cli, name, ns); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "MiddlewareAction %s completed successfully\n", name)
	}
	return nil
}

// parseParams converts []string{"k1=v1", "k2=v2"} into a RawExtension JSON object.
// parseParams 将 key=value 字符串列表转换为 RawExtension JSON 对象。
func parseParams(params []string) (runtime.RawExtension, error) {
	if len(params) == 0 {
		return runtime.RawExtension{}, nil
	}

	m := make(map[string]string, len(params))
	for _, p := range params {
		// Support both "key=value" and comma-separated "k1=v1,k2=v2".
		for _, pair := range strings.Split(p, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				return runtime.RawExtension{}, fmt.Errorf("invalid param %q: expected key=value", pair)
			}
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return runtime.RawExtension{}, err
	}
	return runtime.RawExtension{Raw: raw}, nil
}

// waitForAction polls until the MiddlewareAction reaches a terminal state.
// waitForAction 轮询 MiddlewareAction 直到其进入终态（Available 或失败）。
func waitForAction(ctx context.Context, cli sigs.Client, name, namespace string) error {
	const pollInterval = 2 * time.Second
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for MiddlewareAction %q: %w", name, ctx.Err())
		case <-time.After(pollInterval):
		}

		action := &zeusv1.MiddlewareAction{}
		if err := cli.Get(ctx, sigs.ObjectKey{Name: name, Namespace: namespace}, action); err != nil {
			return fmt.Errorf("get MiddlewareAction: %w", err)
		}

		switch action.Status.State {
		case zeusv1.StateAvailable:
			return nil
		case zeusv1.StateUnavailable:
			// Unavailable is a terminal failure state for one-shot actions.
			// Unavailable 表示一次性 action 执行失败。
			reason := string(action.Status.Reason)
			if reason == "" {
				reason = "unknown"
			}
			return fmt.Errorf("MiddlewareAction %q failed: %s", name, reason)
		}
	}
}
