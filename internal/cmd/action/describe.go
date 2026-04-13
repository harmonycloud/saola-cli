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

	zeusv1 "github.com/OpenSaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// DescribeOptions holds parameters for "action describe".
// DescribeOptions 保存 "action describe" 命令的参数。
type DescribeOptions struct {
	Config    *config.Config
	Name      string
	Namespace string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdDescribe returns the "action describe" command.
//
// 返回 action describe 子命令。
func NewCmdDescribe(cfg *config.Config) *cobra.Command {
	o := &DescribeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "describe <name>",
		Short: lang.T("显示 MiddlewareAction 资源的详细信息", "Show detailed information of a MiddlewareAction resource"),
		Long: lang.T(
			`按名称获取 MiddlewareAction 并以人类可读的格式展示其完整状态和事件信息。`,
			`Fetch a MiddlewareAction by name and display its full status and event details in a human-readable format.`,
		),
		Example: `  saola action describe my-action-1234567890
  saola action describe my-action-1234567890 -n my-namespace`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	return cmd
}

// Run executes the action describe logic.
// Run 执行 action describe 的核心逻辑。
func (o *DescribeOptions) Run(ctx context.Context) error {
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}

	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	action := &zeusv1.MiddlewareAction{}
	if err := cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: ns}, action); err != nil {
		return fmt.Errorf("get MiddlewareAction: %w", err)
	}

	printActionDescribe(action)
	return nil
}

// printActionDescribe prints a human-readable summary of a MiddlewareAction.
// printActionDescribe 打印 MiddlewareAction 的详细可读信息。
func printActionDescribe(a *zeusv1.MiddlewareAction) {
	w := os.Stdout

	fmt.Fprintf(w, "Name:       %s\n", a.Name)
	fmt.Fprintf(w, "Namespace:  %s\n", a.Namespace)
	fmt.Fprintf(w, "Created:    %s\n", a.CreationTimestamp.String())
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Spec:")
	fmt.Fprintf(w, "  Middleware:  %s\n", a.Spec.MiddlewareName)
	fmt.Fprintf(w, "  Baseline:    %s\n", a.Spec.Baseline)
	if len(a.Spec.Necessary.Raw) > 0 {
		formatted, err := formatJSON(a.Spec.Necessary.Raw)
		if err == nil {
			fmt.Fprintf(w, "  Necessary:   %s\n", formatted)
		} else {
			fmt.Fprintf(w, "  Necessary:   %s\n", string(a.Spec.Necessary.Raw))
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Status:")
	state := string(a.Status.State)
	if state == "" {
		state = "<unknown>"
	}
	fmt.Fprintf(w, "  State:   %s\n", state)
	if a.Status.Reason != "" {
		fmt.Fprintf(w, "  Reason:  %s\n", a.Status.Reason)
	}
	if a.Status.ObservedGeneration > 0 {
		fmt.Fprintf(w, "  ObservedGeneration:  %d\n", a.Status.ObservedGeneration)
	}

	if len(a.Status.Conditions) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Conditions:")
		printConditions(w, a.Status.Conditions)
	}
}

// printConditions prints a formatted conditions table.
// printConditions 格式化打印 conditions 列表。
func printConditions(w *os.File, conditions []metav1.Condition) {
	fmt.Fprintf(w, "  %-40s %-8s %-30s %s\n", "TYPE", "STATUS", "REASON", "MESSAGE")
	for _, c := range conditions {
		msg := c.Message
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		fmt.Fprintf(w, "  %-40s %-8s %-30s %s\n", c.Type, c.Status, c.Reason, msg)
	}
}

// formatJSON pretty-prints a raw JSON byte slice.
// formatJSON 将原始 JSON 字节格式化为缩进形式。
func formatJSON(raw []byte) (string, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(v, "    ", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
