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
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee.com/opensaola/saola-cli/internal/client"
	"gitee.com/opensaola/saola-cli/internal/cmdutil"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"gitee.com/opensaola/saola-cli/internal/printer"
	zeusv1 "github.com/opensaola/opensaola/api/v1"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOptions holds parameters for "action get".
// GetOptions 保存 "action get" 命令的参数。
type GetOptions struct {
	Config        *config.Config
	Name          string
	Namespace     string
	AllNamespaces bool
	Output        string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdGet returns the "action get" command.
//
// 返回 action get 子命令。
func NewCmdGet(cfg *config.Config) *cobra.Command {
	o := &GetOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "get [name]",
		Short: lang.T("列出或获取 MiddlewareAction 资源", "List or get MiddlewareAction resources"),
		Long: lang.T(
			`列出命名空间内的所有 MiddlewareAction，或按名称获取单个资源。
使用 -A / --all-namespaces 跨所有命名空间列出。`,
			`List all MiddlewareActions in a namespace, or get a single one by name.
Use -A / --all-namespaces to list across all namespaces.`,
		),
		Example: `  # 列出当前命名空间的所有 action / List all actions in the current namespace
  saola action get

  # 获取指定 action / Get a specific action
  saola action get my-action-1234567890

  # 跨所有命名空间列出 / List across all namespaces
  saola action get -A

  # 以 YAML 格式输出 / Output as YAML
  saola action get my-action-1234567890 -o yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("目标命名空间", "Target namespace"))
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, lang.T("列出所有命名空间中的 action", "List actions across all namespaces"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "table", lang.T("输出格式：table|yaml|json", "Output format: table|yaml|json"))
	return cmd
}

// Run executes the action get logic.
// Run 执行 action get 的核心逻辑。
func (o *GetOptions) Run(ctx context.Context) error {
	ns := o.Namespace
	if ns == "" && !o.AllNamespaces {
		ns = o.Config.Namespace
	}

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

	// Single resource get.
	//
	// 获取单个资源。
	if o.Name != "" {
		action := &zeusv1.MiddlewareAction{}
		if err = cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: ns}, action); err != nil {
			return fmt.Errorf("get MiddlewareAction: %w", err)
		}
		if o.Output == "table" || o.Output == "" {
			return printActionTable([]*zeusv1.MiddlewareAction{action}, o.AllNamespaces)
		}
		return p.Print(os.Stdout, action)
	}

	// List resources.
	//
	// 列出资源列表。
	list := &zeusv1.MiddlewareActionList{}
	var listOpts []sigs.ListOption
	if ns != "" {
		listOpts = append(listOpts, sigs.InNamespace(ns))
	}
	if err = cli.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("list MiddlewareActions: %w", err)
	}

	if len(list.Items) == 0 {
		fmt.Fprintln(os.Stdout, "No MiddlewareActions found.")
		return nil
	}

	if o.Output == "table" || o.Output == "" {
		items := make([]*zeusv1.MiddlewareAction, 0, len(list.Items))
		for i := range list.Items {
			items = append(items, &list.Items[i])
		}
		return printActionTable(items, o.AllNamespaces)
	}
	return p.Print(os.Stdout, list.Items)
}

// printActionTable renders a table with columns: NAME NAMESPACE MIDDLEWARE BASELINE STATE AGE.
// printActionTable 以表格形式输出 NAME NAMESPACE MIDDLEWARE BASELINE STATE AGE 列。
func printActionTable(actions []*zeusv1.MiddlewareAction, showNamespace bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if showNamespace {
		fmt.Fprintln(w, "NAME\tNAMESPACE\tMIDDLEWARE\tBASELINE\tSTATE\tAGE")
	} else {
		fmt.Fprintln(w, "NAME\tMIDDLEWARE\tBASELINE\tSTATE\tAGE")
	}
	for _, a := range actions {
		age := cmdutil.FormatAge(time.Since(a.CreationTimestamp.Time))
		state := string(a.Status.State)
		if state == "" {
			state = "<unknown>"
		}
		if showNamespace {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				a.Name, a.Namespace, a.Spec.MiddlewareName, a.Spec.Baseline, state, age)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				a.Name, a.Spec.MiddlewareName, a.Spec.Baseline, state, age)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	return nil
}

