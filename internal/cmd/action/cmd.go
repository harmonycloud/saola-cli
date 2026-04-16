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
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
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
