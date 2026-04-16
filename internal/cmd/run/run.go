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

package run

// Package run provides the top-level "saola run" command.
//
// run 包提供顶层 "saola run" 命令。

import (
	"github.com/harmonycloud/saola-cli/internal/cmd/action"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewCmdRun returns the top-level "saola run <baseline>" command.
// It delegates execution to action.RunOptions so all core logic lives in one place.
//
// 返回顶层 "saola run <baseline>" 命令。
// 执行逻辑委托给 action.RunOptions，核心逻辑保持单一位置。
func NewCmdRun(cfg *config.Config) *cobra.Command {
	o := &action.RunOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "run <baseline>",
		Short: lang.T("触发一个 MiddlewareAction", "Trigger a MiddlewareAction"),
		Long: lang.T(
			`创建一个 MiddlewareAction CR，对指定 Middleware 实例触发一次性运维操作。
Baseline 作为位置参数传入，Action 名称自动生成为 <baseline>-<unix时间戳> 以避免冲突。`,
			`Create a MiddlewareAction CR to trigger a one-off action against a Middleware instance.
The baseline is passed as a positional argument; the action name is auto-generated
as <baseline>-<unix-timestamp> to avoid conflicts.`,
		),
		Example: `  # 执行备份操作 / Run a backup action
  saola run redis-backup --middleware my-redis

  # 带参数执行并等待最多 5 分钟 / Run with extra parameters and wait up to 5 minutes
  saola run redis-restore --middleware my-redis --params src=backup-001 --wait 5m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Positional arg is the baseline name.
			//
			// 位置参数即 baseline 名称。
			o.Baseline = args[0]
			return o.Run(cmd.Context())
		},
	}

	// Bind flags to RunOptions fields.
	//
	// 将 flag 绑定到 RunOptions 字段。
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间（默认使用配置中的命名空间）", "Target namespace (defaults to config namespace)"))
	cmd.Flags().StringVar(&o.Middleware, "middleware", "", lang.T("关联的 Middleware 实例名（必填）", "Name of the Middleware instance to run the action against (required)"))
	cmd.Flags().StringArrayVar(&o.Params, "params", nil, lang.T("key=value 格式的操作参数，可逗号分隔或重复使用", "Action parameters in key=value format, comma-separated or repeatable"))
	cmd.Flags().DurationVar(&o.Wait, "wait", 0, lang.T("等待操作完成（如 5m，0 表示不等待）", "Wait for the action to complete (e.g. 5m, 0 = don't wait)"))

	_ = cmd.MarkFlagRequired("middleware")

	return cmd
}
