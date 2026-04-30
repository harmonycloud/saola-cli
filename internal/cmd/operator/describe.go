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
	"strings"
	"text/tabwriter"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/client"
	"github.com/harmonycloud/saola-cli/internal/cmdutil"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// DescribeOptions holds parameters for "operator describe".
//
// DescribeOptions 保存 "operator describe" 命令的参数。
type DescribeOptions struct {
	Config    *config.Config
	Name      string
	Namespace string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdDescribe returns the "operator describe" command.
//
// 返回 "operator describe" 子命令。
func NewCmdDescribe(cfg *config.Config) *cobra.Command {
	o := &DescribeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "describe <name>",
		Short: lang.T("显示 MiddlewareOperator 资源的详细信息", "Show detailed information of a MiddlewareOperator resource"),
		Long: lang.T(
			`打印 MiddlewareOperator 的完整 spec、status、conditions 及每个 operator 的 deployment 状态。`,
			`Print the full spec, status, conditions, and per-operator deployment status of a MiddlewareOperator.`,
		),
		Example: `  saola operator describe redis-operator -n my-ns`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	return cmd
}

// Run executes the describe logic.
//
// Run 执行 describe 逻辑。
func (o *DescribeOptions) Run(ctx context.Context) error {
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}
	// MiddlewareOperator requires an explicit namespace to prevent accidental operations
	// across namespaces, as operators manage cluster-wide middleware types.
	//
	// MiddlewareOperator 要求显式指定 namespace，以防止跨 namespace 的误操作，因为 operator 管理的是集群级别的中间件类型。
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

	mo := &zeusv1.MiddlewareOperator{}
	if err := cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: ns}, mo); err != nil {
		return fmt.Errorf("get MiddlewareOperator %s/%s: %w", ns, o.Name, err)
	}

	printDescribe(os.Stdout, mo)
	return nil
}

// printDescribe writes formatted describe output.
//
// printDescribe 格式化输出 describe 信息。
func printDescribe(w *os.File, mo *zeusv1.MiddlewareOperator) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer func() {
		_ = tw.Flush()
	}()

	// Basic metadata section.
	//
	// 基础元数据部分。
	fmt.Fprintf(tw, "Name:\t%s\n", mo.Name)
	fmt.Fprintf(tw, "Namespace:\t%s\n", mo.Namespace)
	fmt.Fprintf(tw, "Labels:\t%s\n", formatLabels(mo.Labels))
	fmt.Fprintf(tw, "Annotations:\t%s\n", formatLabels(mo.Annotations))
	if !mo.CreationTimestamp.IsZero() {
		fmt.Fprintf(tw, "Created:\t%s\n", mo.CreationTimestamp.Format("2006-01-02T15:04:05Z"))
	}

	// Spec section.
	//
	// Spec 部分。
	fmt.Fprintf(tw, "\nSpec:\n")
	fmt.Fprintf(tw, "  Baseline:\t%s\n", mo.Spec.Baseline)
	fmt.Fprintf(tw, "  PermissionScope:\t%s\n", string(mo.Spec.PermissionScope))
	if len(mo.Spec.Configurations) > 0 {
		fmt.Fprintf(tw, "  Configurations:\t%d item(s)\n", len(mo.Spec.Configurations))
		for _, cfg := range mo.Spec.Configurations {
			fmt.Fprintf(tw, "    - %s\n", cfg.Name)
		}
	}
	if len(mo.Spec.PreActions) > 0 {
		fmt.Fprintf(tw, "  PreActions:\t%d item(s)\n", len(mo.Spec.PreActions))
		for _, pa := range mo.Spec.PreActions {
			fmt.Fprintf(tw, "    - %s (fixed=%v exposed=%v)\n", pa.Name, pa.Fixed, pa.Exposed)
		}
	}

	// Status section.
	//
	// Status 部分。
	fmt.Fprintf(tw, "\nStatus:\n")
	fmt.Fprintf(tw, "  State:\t%s\n", string(mo.Status.State))
	fmt.Fprintf(tw, "  Ready:\t%v\n", mo.Status.Ready)
	fmt.Fprintf(tw, "  Runtime:\t%s\n", mo.Status.Runtime)
	fmt.Fprintf(tw, "  OperatorAvailable:\t%s\n", mo.Status.OperatorAvailable)
	if mo.Status.Reason != "" {
		fmt.Fprintf(tw, "  Reason:\t%s\n", mo.Status.Reason)
	}
	fmt.Fprintf(tw, "  ObservedGeneration:\t%d\n", mo.Status.ObservedGeneration)

	// Conditions section.
	//
	// Conditions 部分。
	if len(mo.Status.Conditions) > 0 {
		fmt.Fprintf(tw, "\nConditions:\n")
		fmt.Fprintf(tw, "  %-30s\t%-8s\t%-30s\t%s\n", "TYPE", "STATUS", "REASON", "MESSAGE")
		for _, c := range mo.Status.Conditions {
			fmt.Fprintf(tw, "  %-30s\t%-8s\t%-30s\t%s\n",
				c.Type,
				string(c.Status),
				c.Reason,
				cmdutil.Truncate(c.Message, 60),
			)
		}
	}

	// OperatorStatus section (per-deployment details).
	//
	// OperatorStatus 部分（每个 Deployment 的详细状态）。
	if len(mo.Status.OperatorStatus) > 0 {
		fmt.Fprintf(tw, "\nOperatorStatus:\n")
		for name, ds := range mo.Status.OperatorStatus {
			fmt.Fprintf(tw, "  %s:\n", name)
			printDeploymentStatus(tw, ds)
		}
	}

	// Finalizers.
	//
	// Finalizer 列表。
	if len(mo.Finalizers) > 0 {
		fmt.Fprintf(tw, "\nFinalizers:\t%s\n", strings.Join(mo.Finalizers, ", "))
	}
}

// printDeploymentStatus prints key fields from a DeploymentStatus.
//
// printDeploymentStatus 打印 DeploymentStatus 的关键字段。
func printDeploymentStatus(tw *tabwriter.Writer, ds appsv1.DeploymentStatus) {
	fmt.Fprintf(tw, "    Replicas:\t%d\n", ds.Replicas)
	fmt.Fprintf(tw, "    Ready:\t%d\n", ds.ReadyReplicas)
	fmt.Fprintf(tw, "    Available:\t%d\n", ds.AvailableReplicas)
	fmt.Fprintf(tw, "    Updated:\t%d\n", ds.UpdatedReplicas)
	if len(ds.Conditions) > 0 {
		fmt.Fprintf(tw, "    Conditions:\n")
		for _, c := range ds.Conditions {
			fmt.Fprintf(tw, "      %s=%s (%s)\n", c.Type, c.Status, c.Reason)
		}
	}
}

// formatLabels converts a label/annotation map to a sorted key=value string.
//
// formatLabels 将 label/annotation map 转换为 key=value 字符串。
func formatLabels(m map[string]string) string {
	if len(m) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ", ")
}
