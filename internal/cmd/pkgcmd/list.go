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

package pkgcmd

import (
	"context"
	"fmt"
	"os"

	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/printer"
	"gitee.com/opensaola/saola-cli/internal/packages"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListOptions holds parameters for "package list".
// ListOptions 保存 "package list" 命令的参数。
type ListOptions struct {
	Config    *config.Config
	Component string
	Version   string
	Output    string
	// Client is the k8s client to use. If nil, a real client is built from Config.
	//
	// Client 为注入的 k8s 客户端；为 nil 时从 Config 构建真实客户端（供测试注入 fake client）。
	Client sigs.Client
}

// NewCmdList returns the "package list" command.
//
// 返回 package list 子命令。
func NewCmdList(cfg *config.Config) *cobra.Command {
	o := &ListOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short: lang.T("列出已安装的中间件包", "List installed middleware packages"),
		Long: lang.T(
			`列出 pkg-namespace 中所有已安装的中间件包 Secret，支持按组件名和版本过滤。`,
			`List all installed middleware package Secrets in the pkg-namespace. Supports filtering by component name and version.`,
		),
		Example: `  saola package list
  saola package list --component redis
  saola package list -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&o.Component, "component", "", lang.T("按组件名过滤", "Filter by component name"))
	cmd.Flags().StringVar(&o.Version, "version", "", lang.T("按包版本过滤", "Filter by package version"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	return cmd
}

// pkgRow is a display-friendly row for table output.
// pkgRow 是表格输出使用的展示行结构。
type pkgRow struct {
	Name      string
	Component string
	Version   string
	Enabled   string
	Created   string
}

func (o *ListOptions) Run(ctx context.Context) error {
	// Set the packages data namespace so opensaola helper functions target the right namespace.
	packages.SetDataNamespace(o.Config.PkgNamespace)

	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	pkgs, err := packages.List(ctx, cli, packages.Option{
		LabelComponent:      o.Component,
		LabelPackageVersion: o.Version,
	})
	if err != nil {
		return fmt.Errorf("list packages: %w", err)
	}

	if len(pkgs) == 0 {
		fmt.Fprintln(os.Stdout, "No packages found.")
		return nil
	}

	p, err := printer.New(o.Output)
	if err != nil {
		return err
	}

	switch o.Output {
	case "yaml", "json":
		return p.Print(os.Stdout, pkgs)
	default:
		rows := make([]pkgRow, 0, len(pkgs))
		for _, pkg := range pkgs {
			enabled := "false"
			if pkg.Enabled {
				enabled = "true"
			}
			ver := ""
			if pkg.Metadata != nil {
				ver = pkg.Metadata.Version
			}
			rows = append(rows, pkgRow{
				Name:      pkg.Name,
				Component: pkg.Component,
				Version:   ver,
				Enabled:   enabled,
				Created:   pkg.Created,
			})
		}
		return p.Print(os.Stdout, rows)
	}
}
