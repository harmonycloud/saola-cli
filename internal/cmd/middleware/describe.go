package middleware

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	zeusk8s "gitee.com/opensaola/opensaola/pkg/k8s"
	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// DescribeOptions holds parameters for "middleware describe".
//
// DescribeOptions 保存 middleware describe 子命令的所有参数。
type DescribeOptions struct {
	Config    *config.Config
	Name      string
	Namespace string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；为 nil 时使用 client.New(cfg)。
	Client sigs.Client
}

// NewCmdDescribe returns the middleware describe command.
//
// 返回 middleware describe 子命令。
func NewCmdDescribe(cfg *config.Config) *cobra.Command {
	o := &DescribeOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "describe <name>",
		Short: lang.T("显示 Middleware 资源的详细信息", "Show detailed information about a Middleware resource"),
		Long: lang.T(
			`获取单个 Middleware 资源并以可读格式输出其 spec、status 和 conditions。`,
			`Fetch a single Middleware and print its spec, status and conditions in human-readable form.`,
		),
		Example: `  saola middleware describe my-redis
  saola middleware describe my-redis -n production`,
		Args: cobra.ExactArgs(1),
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
// 执行 describe 逻辑：获取单个对象并以 human-readable 格式输出 spec、status 和 conditions。
func (o *DescribeOptions) Run(ctx context.Context) error {
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}
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

	mw, err := zeusk8s.GetMiddleware(ctx, cli, o.Name, ns)
	if err != nil {
		return fmt.Errorf("get middleware %s/%s: %w", ns, o.Name, err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// --- Metadata ---
	//
	// --- 基础元信息 ---
	fmt.Fprintf(w, "Name:\t%s\n", mw.Name)
	fmt.Fprintf(w, "Namespace:\t%s\n", mw.Namespace)
	fmt.Fprintf(w, "Age:\t%s\n", formatAge(mw.CreationTimestamp.Time))
	if len(mw.Labels) > 0 {
		fmt.Fprintf(w, "Labels:\t%s\n", formatLabels(mw.Labels))
	}
	if len(mw.Annotations) > 0 {
		fmt.Fprintf(w, "Annotations:\t%s\n", formatAnnotations(mw.Annotations))
	}

	// --- Spec ---
	//
	// --- 期望状态 ---
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Spec:")
	fmt.Fprintf(w, "  Baseline:\t%s\n", mw.Spec.Baseline)
	if mw.Spec.OperatorBaseline.Name != "" || mw.Spec.OperatorBaseline.GvkName != "" {
		fmt.Fprintln(w, "  OperatorBaseline:")
		fmt.Fprintf(w, "    Name:\t%s\n", mw.Spec.OperatorBaseline.Name)
		fmt.Fprintf(w, "    GvkName:\t%s\n", mw.Spec.OperatorBaseline.GvkName)
	}
	if len(mw.Spec.Configurations) > 0 {
		fmt.Fprintf(w, "  Configurations:\t%d item(s)\n", len(mw.Spec.Configurations))
		for _, c := range mw.Spec.Configurations {
			fmt.Fprintf(w, "    - Name:\t%s\n", c.Name)
		}
	}
	if len(mw.Spec.PreActions) > 0 {
		fmt.Fprintf(w, "  PreActions:\t%d item(s)\n", len(mw.Spec.PreActions))
	}

	// --- Status ---
	//
	// --- 实际状态 ---
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Status:")
	fmt.Fprintf(w, "  State:\t%s\n", mw.Status.State)
	if mw.Status.Reason != "" {
		fmt.Fprintf(w, "  Reason:\t%s\n", mw.Status.Reason)
	}
	fmt.Fprintf(w, "  ObservedGeneration:\t%d\n", mw.Status.ObservedGeneration)

	cr := mw.Status.CustomResources
	if cr.Phase != "" || cr.Type != "" || cr.Replicas > 0 {
		fmt.Fprintln(w, "  CustomResources:")
		if cr.Type != "" {
			fmt.Fprintf(w, "    Type:\t%s\n", cr.Type)
		}
		if cr.Phase != "" {
			fmt.Fprintf(w, "    Phase:\t%s\n", cr.Phase)
		}
		if cr.Replicas > 0 {
			fmt.Fprintf(w, "    Replicas:\t%d\n", cr.Replicas)
		}
		if cr.Reason != "" {
			fmt.Fprintf(w, "    Reason:\t%s\n", cr.Reason)
		}
	}

	// --- Conditions ---
	//
	// --- 状态条件 ---
	if len(mw.Status.Conditions) > 0 {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Conditions:")
		fmt.Fprintf(w, "  %-20s\t%-8s\t%-10s\t%s\n", "TYPE", "STATUS", "REASON", "MESSAGE")
		for _, c := range mw.Status.Conditions {
			fmt.Fprintf(w, "  %-20s\t%-8s\t%-10s\t%s\n",
				c.Type,
				conditionStatus(c),
				c.Reason,
				truncate(c.Message, 60),
			)
		}
	}

	return nil
}

// conditionStatus returns the Status field of a metav1.Condition as a string.
//
// 返回 metav1.Condition 的 Status 字符串。
func conditionStatus(c metav1.Condition) string {
	return string(c.Status)
}

// formatLabels renders a label map as "k=v,k=v".
//
// 将 label map 格式化为 "k=v,k=v" 字符串。
func formatLabels(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

// formatAnnotations renders an annotation map, skipping large values.
//
// 格式化 annotation map，跳过过长的值。
func formatAnnotations(annotations map[string]string) string {
	parts := make([]string, 0, len(annotations))
	for k, v := range annotations {
		parts = append(parts, k+"="+truncate(v, 40))
	}
	return strings.Join(parts, ",")
}

// truncate shortens s to maxLen characters, appending "..." if needed.
//
// 将字符串截断到 maxLen 个字符，超出时追加 "..."。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
