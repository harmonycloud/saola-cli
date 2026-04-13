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
