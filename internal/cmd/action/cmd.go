package action

import (
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdAction returns the "action" sub-command group.
//
// 返回 action 子命令组。
func NewCmdAction(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action",
		Short: lang.T("管理 MiddlewareAction 资源", "Manage MiddlewareAction resources"),
		Long: lang.T(
			`运行、查询并描述 MiddlewareAction 资源。`,
			`Run, get and describe MiddlewareAction resources.`,
		),
	}

	cmd.AddCommand(
		NewCmdRun(cfg),
		NewCmdGet(cfg),
		NewCmdDescribe(cfg),
	)
	return cmd
}
