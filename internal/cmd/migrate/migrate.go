// Copyright 2026 The OpenSaola Authors. SPDX-License-Identifier: Apache-2.0
package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	appclient "github.com/harmonycloud/saola-cli/internal/client"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/harmonycloud/saola-cli/internal/ssamigrate"
	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
)

// NewCmdMigrate adds explicit, resumable metadata migration commands.
// NewCmdMigrate 提供显式、可恢复的 metadata 迁移命令。
func NewCmdMigrate(cfg *config.Config) *cobra.Command {
	parent := &cobra.Command{Use: "migrate", Short: lang.T("迁移历史资源管理信息", "Migrate historical resource ownership")}
	ssa := &cobra.Command{Use: "ssa", Short: lang.T("迁移 OpenSaola 历史 SSA 字段归属", "Migrate legacy OpenSaola SSA field ownership")}
	parent.AddCommand(ssa)
	var auditFile, schemaFile, writer, output string
	var all bool
	plan := &cobra.Command{Use: "plan", Args: cobra.NoArgs, Short: lang.T("只读扫描并生成迁移计划", "Scan read-only and generate a migration plan"), RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := newEngine(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		var history *ssamigrate.Schemas
		var events []ssamigrate.AuditEvent
		if schemaFile != "" {
			raw, err := os.ReadFile(schemaFile)
			if err != nil {
				return err
			}
			history, err = ssamigrate.ReadSchemas(raw)
			if err != nil {
				return err
			}
		}
		if auditFile != "" {
			f, err := os.Open(auditFile)
			if err != nil {
				return err
			}
			defer f.Close()
			events, err = ssamigrate.ReadAudit(f)
			if err != nil {
				return err
			}
		}
		ns := cfg.Namespace
		if all {
			ns = ""
		}
		var missing []string
		if writer == "" {
			missing = append(missing, "--writer")
		}
		if auditFile == "" {
			missing = append(missing, "--audit-log")
		}
		if schemaFile == "" {
			missing = append(missing, "--historical-openapi")
		}
		if len(missing) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", lang.T(
				fmt.Sprintf("警告：未提供 %s，无法建立字段来源证据；计划条目将全部 blocked 且不可执行。补齐历史资料后重跑以生成可执行计划。", strings.Join(missing, "、")),
				fmt.Sprintf("warning: %s not provided; provenance cannot be established, so all entries will be blocked and non-executable. Re-run with the historical inputs to produce an executable plan.", strings.Join(missing, ", "))))
		}
		p, err := engine.Scan(cmd.Context(), ns, writer, events, history)
		if err != nil {
			return err
		}
		if output != "" {
			if err = ssamigrate.SaveExclusive(output, p); err != nil {
				return err
			}
		}
		if err = json.NewEncoder(cmd.OutOrStdout()).Encode(p); err != nil {
			return err
		}
		if len(p.Errors) > 0 {
			return fmt.Errorf("scan incomplete; see errors in plan")
		}
		return nil
	}}
	plan.Flags().StringVar(&auditFile, "audit-log", "", lang.T("历史 RequestResponse 审计 JSONL", "Historical RequestResponse audit JSONL"))
	plan.Flags().StringVar(&schemaFile, "historical-openapi", "", lang.T("创建期 OpenAPI v2 JSON", "OpenAPI v2 JSON captured at creation"))
	plan.Flags().StringVar(&writer, "writer", "", lang.T("可信 OpenSaola 审计用户名（通常为 ServiceAccount）", "Trusted OpenSaola audit username (usually a ServiceAccount)"))
	plan.Flags().StringVarP(&output, "output", "o", "", lang.T("独占写入计划文件", "Write an exclusive plan file"))
	plan.Flags().BoolVarP(&all, "all-namespaces", "A", false, lang.T("扫描所有命名空间", "Scan all namespaces"))
	ssa.AddCommand(plan)
	for _, verb := range []string{"apply", "verify"} {
		var path, backup string
		cmd := &cobra.Command{Use: verb, Args: cobra.NoArgs, Short: lang.T(map[string]string{"apply": "执行计划，仅修改字段归属", "verify": "校验字段归属迁移结果"}[verb], map[string]string{"apply": "Execute metadata-only migration", "verify": "Verify ownership migration"}[verb]), RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var p ssamigrate.Plan
			if err = json.Unmarshal(raw, &p); err != nil {
				return err
			}
			engine, err := newEngine(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			var result []ssamigrate.Result
			if verb == "apply" {
				result, err = engine.Apply(cmd.Context(), &p, backup)
			} else {
				result, err = engine.Verify(cmd.Context(), &p)
			}
			if result != nil {
				if outputErr := json.NewEncoder(cmd.OutOrStdout()).Encode(result); outputErr != nil {
					return outputErr
				}
			}
			return err
		}}
		cmd.Flags().StringVar(&path, "plan", "", lang.T("plan 生成的原始 JSON 文件", "Original JSON file produced by plan"))
		_ = cmd.MarkFlagRequired("plan")
		if verb == "apply" {
			cmd.Flags().StringVar(&backup, "backup-dir", "", lang.T("原始资源备份目录（0700）", "Original resource backup directory (0700)"))
			_ = cmd.MarkFlagRequired("backup-dir")
		}
		ssa.AddCommand(cmd)
	}
	ssa.PersistentFlags().Duration("timeout", 2*time.Minute, lang.T("命令执行超时", "Command timeout"))
	for _, c := range ssa.Commands() {
		run := c.RunE
		c.RunE = func(cmd *cobra.Command, args []string) error {
			d, _ := cmd.Flags().GetDuration("timeout")
			if d <= 0 {
				return fmt.Errorf("timeout must be positive")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), d)
			defer cancel()
			cmd.SetContext(ctx)
			return run(cmd, args)
		}
	}
	return parent
}
func newEngine(ctx context.Context, cfg *config.Config) (*ssamigrate.Engine, error) {
	wrapper := appclient.New(cfg)
	cli, err := wrapper.Get()
	if err != nil {
		return nil, err
	}
	rest, err := wrapper.RESTConfig()
	if err != nil {
		return nil, err
	}
	rest.Timeout = 30 * time.Second
	dc, err := discovery.NewDiscoveryClientForConfig(rest)
	if err != nil {
		return nil, err
	}
	fetchSchemas := func(ctx context.Context) (*ssamigrate.Schemas, error) {
		raw, err := dc.RESTClient().Get().AbsPath("/openapi/v2").SetHeader("Accept", "application/json").DoRaw(ctx)
		if err != nil {
			return nil, err
		}
		return ssamigrate.ReadSchemas(raw)
	}
	schemas, err := fetchSchemas(ctx)
	if err != nil {
		return nil, err
	}
	return &ssamigrate.Engine{Client: cli, Discovery: dc, Current: schemas, PackageNamespace: cfg.PkgNamespace, RefreshSchemas: fetchSchemas}, nil
}
