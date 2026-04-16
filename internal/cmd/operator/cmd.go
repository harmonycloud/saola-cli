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
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
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
		NewCmdUpgrade(cfg),
	)
	return cmd
}
