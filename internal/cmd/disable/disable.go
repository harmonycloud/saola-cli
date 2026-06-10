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

package disable

import (
	"github.com/harmonycloud/saola-cli/internal/cmd/pkgcmd"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdDisable returns the top-level "disable" command.
//
// NewCmdDisable 返回顶层 disable 命令。
func NewCmdDisable(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable",
		Short: lang.T("禁用资源或包", "Disable resources or packages"),
		Long: lang.T(
			`禁用资源或包。当前支持禁用中间件包，不删除包 Secret。`,
			`Disable resources or packages. Currently supports disabling middleware packages without deleting package Secrets.`,
		),
	}
	cmd.AddCommand(pkgcmd.NewCmdDisablePackage(cfg))
	return cmd
}
