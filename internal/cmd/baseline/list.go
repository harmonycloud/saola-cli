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
	"context"
	"fmt"
	"os"

	zeusv1 "gitee.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/printer"
	"gitee.com/opensaola/saola-cli/internal/packages"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListOptions holds parameters for "baseline list".
// ListOptions 保存 "baseline list" 命令的参数。
type ListOptions struct {
	Config  *config.Config
	Package string
	Kind    string
	Output  string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdList returns the "baseline list" command.
//
// 返回 baseline list 子命令。
func NewCmdList(cfg *config.Config) *cobra.Command {
	o := &ListOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short: lang.T("列出已安装包中的 baseline", "List baselines in an installed package"),
		Long: lang.T(
			`列出指定已安装包中所有 MiddlewareBaseline 或 MiddlewareOperatorBaseline。
使用 --kind 指定 baseline 类型（middleware 或 operator）。`,
			`List all MiddlewareBaseline or MiddlewareOperatorBaseline resources in an installed package.
Use --kind to specify the baseline type (middleware or operator).`,
		),
		Example: `  saola baseline list --package redis-v1
  saola baseline list --package redis-v1 --kind middleware`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&o.Package, "package", "", lang.T("要查询的包名（必填）", "Package name to query (required)"))
	cmd.Flags().StringVar(&o.Kind, "kind", "middleware", lang.T("Baseline 类型：middleware|operator|action", "Baseline kind: middleware|operator|action"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	_ = cmd.MarkFlagRequired("package")
	return cmd
}

func (o *ListOptions) Run(ctx context.Context) error {
	packages.SetDataNamespace(o.Config.PkgNamespace)

	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	p, err := printer.New(o.Output)
	if err != nil {
		return err
	}

	switch o.Kind {
	case "middleware":
		baselines, err := packages.GetMiddlewareBaselines(ctx, cli, o.Package)
		if err != nil {
			return fmt.Errorf("list middleware baselines: %w", err)
		}
		if len(baselines) == 0 {
			fmt.Fprintln(os.Stdout, "No middleware baselines found.")
			return nil
		}
		if o.Output == "table" {
			return p.Print(os.Stdout, toMwBaselineRows(baselines))
		}
		return p.Print(os.Stdout, baselines)
	case "operator":
		baselines, err := packages.GetMiddlewareOperatorBaselines(ctx, cli, o.Package)
		if err != nil {
			return fmt.Errorf("list operator baselines: %w", err)
		}
		if len(baselines) == 0 {
			fmt.Fprintln(os.Stdout, "No operator baselines found.")
			return nil
		}
		if o.Output == "table" {
			return p.Print(os.Stdout, toOpBaselineRows(baselines))
		}
		return p.Print(os.Stdout, baselines)
	case "action":
		baselines, err := packages.GetMiddlewareActionBaselines(ctx, cli, o.Package)
		if err != nil {
			return fmt.Errorf("list action baselines: %w", err)
		}
		if len(baselines) == 0 {
			fmt.Fprintln(os.Stdout, "No action baselines found.")
			return nil
		}
		if o.Output == "table" {
			return p.Print(os.Stdout, toActionBaselineRows(baselines))
		}
		return p.Print(os.Stdout, baselines)
	default:
		return fmt.Errorf("unknown baseline kind %q (supported: middleware, operator, action)", o.Kind)
	}
}

// baselineRow is a display-friendly table row for baselines.
//
// baselineRow 是 baseline 表格显示的行结构。
type baselineRow struct {
	NAME            string
	OPERATOR        string
	CONFIGURATIONS  int
	PREACTIONS      int
}

// toMwBaselineRows converts MiddlewareBaseline list to table rows.
//
// toMwBaselineRows 将 MiddlewareBaseline 列表转换为表格行。
func toMwBaselineRows(baselines []*zeusv1.MiddlewareBaseline) []baselineRow {
	rows := make([]baselineRow, 0, len(baselines))
	for _, b := range baselines {
		rows = append(rows, baselineRow{
			NAME:           b.Name,
			OPERATOR:       b.Spec.OperatorBaseline.Name,
			CONFIGURATIONS: len(b.Spec.Configurations),
			PREACTIONS:     len(b.Spec.PreActions),
		})
	}
	return rows
}

// toOpBaselineRows converts MiddlewareOperatorBaseline list to table rows.
//
// toOpBaselineRows 将 MiddlewareOperatorBaseline 列表转换为表格行。
func toOpBaselineRows(baselines []*zeusv1.MiddlewareOperatorBaseline) []baselineRow {
	rows := make([]baselineRow, 0, len(baselines))
	for _, b := range baselines {
		rows = append(rows, baselineRow{
			NAME:           b.Name,
			CONFIGURATIONS: len(b.Spec.Configurations),
			PREACTIONS:     len(b.Spec.PreActions),
		})
	}
	return rows
}

// toActionBaselineRows converts MiddlewareActionBaseline list to table rows.
//
// toActionBaselineRows 将 MiddlewareActionBaseline 列表转换为表格行。
func toActionBaselineRows(baselines []*zeusv1.MiddlewareActionBaseline) []baselineRow {
	rows := make([]baselineRow, 0, len(baselines))
	for _, b := range baselines {
		rows = append(rows, baselineRow{
			NAME: b.Name,
		})
	}
	return rows
}
