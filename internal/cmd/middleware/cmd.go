package middleware

import (
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdMiddleware returns the "middleware" sub-command group.
func NewCmdMiddleware(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "middleware",
		Aliases: []string{"mw"},
		Short: lang.T("管理 Middleware 资源", "Manage Middleware resources"),
		Long: lang.T(
			`创建、删除、查询和描述 Middleware 自定义资源。`,
			`Create, delete, get and describe Middleware custom resources.`,
		),
	}

	cmd.AddCommand(
		NewCmdCreate(cfg),
		NewCmdDelete(cfg),
		NewCmdGet(cfg),
		NewCmdDescribe(cfg),
		NewCmdUpgrade(cfg),
	)
	return cmd
}
