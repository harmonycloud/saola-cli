package pkgcmd

import (
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdPackage returns the "package" sub-command group.
//
// 返回 package 子命令组。
func NewCmdPackage(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "package",
		Aliases: []string{"pkg"},
		Short: lang.T("管理中间件包", "Manage middleware packages"),
		Long: lang.T(
			`安装、卸载、升级、列出、检查并构建中间件包。`,
			`Install, uninstall, upgrade, list, inspect and build middleware packages.`,
		),
	}

	cmd.AddCommand(
		NewCmdInstall(cfg),
		NewCmdUninstall(cfg),
		NewCmdUpgrade(cfg),
		NewCmdList(cfg),
		NewCmdInspect(cfg),
		NewCmdBuild(cfg),
	)
	return cmd
}
