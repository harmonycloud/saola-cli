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

package middleware

import (
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdMiddleware returns the "middleware" sub-command group.
func NewCmdMiddleware(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "middleware",
		Aliases: []string{"mw"},
		Short:   lang.T("管理 Middleware 资源", "Manage Middleware resources"),
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
