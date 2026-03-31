package operator

import (
	"gitea.com/middleware-management/saola-cli/internal/config"
	"gitea.com/middleware-management/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdOperator returns the "operator" sub-command group.
//
// 返回 "operator" 子命令组。
func NewCmdOperator(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: lang.T("管理 MiddlewareOperator 资源", "Manage MiddlewareOperator resources"),
		Long: lang.T(
			`创建、删除、查询和描述 MiddlewareOperator 自定义资源。`,
			`Create, delete, get and describe MiddlewareOperator custom resources.`,
		),
	}

	cmd.AddCommand(
		NewCmdCreate(cfg),
		NewCmdDelete(cfg),
		NewCmdGet(cfg),
		NewCmdDescribe(cfg),
	)
	return cmd
}
