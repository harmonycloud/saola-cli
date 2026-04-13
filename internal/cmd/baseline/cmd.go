package baseline

import (
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdBaseline returns the "baseline" sub-command group.
//
// 返回 baseline 子命令组。
func NewCmdBaseline(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: lang.T("查询已安装包中的 baseline", "Query baselines in installed packages"),
		Long: lang.T(
			`列出并获取包中内嵌的 MiddlewareBaseline / MiddlewareOperatorBaseline 资源。`,
			`List and get MiddlewareBaseline / MiddlewareOperatorBaseline resources embedded in a package.`,
		),
	}

	cmd.AddCommand(
		NewCmdList(cfg),
		NewCmdGet(cfg),
	)
	return cmd
}
