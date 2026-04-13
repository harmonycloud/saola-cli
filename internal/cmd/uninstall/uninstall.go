package uninstall

import (
	"gitee.com/opensaola/saola-cli/internal/cmd/pkgcmd"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdUninstall returns the top-level "uninstall" command.
// It is a thin wrapper around pkgcmd.UninstallOptions and delegates all logic to its Run method.
//
// NewCmdUninstall 返回顶层 uninstall 命令。
// 作为 pkgcmd.UninstallOptions 的薄封装，所有逻辑委托给其 Run 方法。
func NewCmdUninstall(cfg *config.Config) *cobra.Command {
	o := &pkgcmd.UninstallOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "uninstall <name>",
		Short: lang.T("卸载中间件包", "Uninstall a middleware package"),
		Long: lang.T(
			`在包对应的 Secret 上添加卸载注解。
zeus-operator 检测到注解后会自动卸载该包。`,
			`Add the uninstall annotation to the package Secret.
zeus-operator will pick up the annotation and uninstall the package.`,
		),
		Example: `  saola uninstall redis-v1
  saola uninstall redis-v1 --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待卸载完成（0 表示不等待）", "Wait for uninstallation to complete (0 = don't wait)"))

	return cmd
}
