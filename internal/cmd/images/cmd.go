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

package images

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	imageexport "github.com/harmonycloud/saola-cli/internal/images"

	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// ExportOptions holds parameters for "images export".
//
// ExportOptions 保存 "images export" 命令参数。
type ExportOptions struct {
	Config       *config.Config
	PkgDir       string
	Output       string
	LockFile     string
	Repositories []string
	Platform     string
	MultiArch    bool
	Insecure     bool
	SkipMissing  bool
	DryRun       bool
	Timeout      time.Duration
	Runner       imageexport.Runner
	Out          io.Writer
	ErrOut       io.Writer
}

// NewCmdImages returns the "images" command group.
//
// NewCmdImages 返回 images 命令组。
func NewCmdImages(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: lang.T("发现并导出包依赖的容器镜像", "Discover and export package container images"),
		Long: lang.T(
			`发现本地中间件包中的镜像引用，并从一个或多个候选仓库中导出实际存在的镜像。`,
			`Discover image references in a local middleware package and export existing images from one or more candidate repositories.`,
		),
	}
	cmd.AddCommand(NewCmdExport(cfg))
	return cmd
}

// NewCmdExport returns the "images export" command.
//
// NewCmdExport 返回 images export 子命令。
func NewCmdExport(cfg *config.Config) *cobra.Command {
	o := &ExportOptions{
		Config:    cfg,
		Platform:  "all",
		MultiArch: true,
		Timeout:   30 * time.Second,
		Out:       os.Stdout,
		ErrOut:    os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "export <pkg-dir>",
		Short: lang.T("导出包依赖的容器镜像", "Export package container images"),
		Long: lang.T(
			`扫描本地中间件包中的镜像引用，按 --repository 的声明顺序探测镜像是否存在，并把命中的镜像导出为归档文件。`,
			`Scan image references in a local middleware package, probe repositories in --repository order, and export the resolved images as an archive.`,
		),
		Example: `  saola images export ./redis -r 10.10.101.172:443/middleware -r 10.10.102.124:443/middleware
  saola images export ./redis -r 10.10.101.172:443/middleware,10.10.102.124:443/middleware -o redis-images.tar
  saola images export ./redis -r 10.10.101.172:443/middleware --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			o.Out = cmd.OutOrStdout()
			o.ErrOut = cmd.ErrOrStderr()
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringArrayVarP(&o.Repositories, "repository", "r", nil, lang.T("候选镜像仓库，可重复声明或用逗号分隔", "Candidate image repository; repeat or comma-separate values"))
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", lang.T("输出镜像归档路径（默认：<name>-<version>-images.tar）", "Output image archive path (default: <name>-<version>-images.tar)"))
	cmd.Flags().StringVar(&o.LockFile, "lock-file", "", lang.T("输出镜像锁定清单路径（默认：<output>.lock.json）", "Output image lock file path (default: <output>.lock.json)"))
	cmd.Flags().StringVar(&o.Platform, "platform", "all", lang.T("导出平台，例如 linux/amd64；all 表示保留多架构", "Export platform, for example linux/amd64; all keeps multi-arch images"))
	cmd.Flags().BoolVar(&o.MultiArch, "multi-arch", true, lang.T("使用 skopeo 导出时保留多架构清单", "Keep multi-arch manifests when exporting with skopeo"))
	cmd.Flags().BoolVar(&o.Insecure, "insecure", false, lang.T("跳过镜像仓库 TLS 校验", "Skip registry TLS verification"))
	cmd.Flags().BoolVar(&o.SkipMissing, "skip-missing", false, lang.T("存在无法解析的镜像时仍导出已命中的镜像", "Export resolved images even when some images are missing"))
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, lang.T("仅打印镜像候选，不执行导出", "Print image candidates without exporting"))
	cmd.Flags().DurationVar(&o.Timeout, "timeout", 30*time.Second, lang.T("单个镜像探测或导出的超时时间", "Timeout for each image probe or export"))
	return cmd
}

// Run executes image export.
//
// Run 执行镜像导出。
func (o *ExportOptions) Run(ctx context.Context) error {
	out := o.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := o.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}

	result, err := imageexport.ExportPackage(ctx, imageexport.ExportOptions{
		PkgDir:       o.PkgDir,
		Output:       o.Output,
		LockFile:     o.LockFile,
		Repositories: o.Repositories,
		Platform:     o.Platform,
		MultiArch:    o.MultiArch,
		Insecure:     o.Insecure,
		SkipMissing:  o.SkipMissing,
		DryRun:       o.DryRun,
		Timeout:      o.Timeout,
		Runner:       o.Runner,
	})
	if err != nil {
		return err
	}

	if o.DryRun {
		printDryRun(out, result)
		return nil
	}

	fmt.Fprintf(out, "Exported %d images for %s@%s -> %s\n", len(result.Resolved), result.Metadata.Name, result.Metadata.Version, result.Output)
	fmt.Fprintf(out, "Wrote lock file -> %s\n", result.LockPath)
	if len(result.Missing) > 0 {
		fmt.Fprintf(errOut, "Skipped %d missing images because --skip-missing was set\n", len(result.Missing))
	}
	return nil
}

func printDryRun(out io.Writer, result *imageexport.ExportResult) {
	fmt.Fprintf(out, "Found %d image groups for %s@%s\n", len(result.Groups), result.Metadata.Name, result.Metadata.Version)
	for _, group := range result.Groups {
		fmt.Fprintf(out, "- %s\n", group.Name)
		for _, candidate := range group.Candidates {
			parts := []string{candidate.Image}
			if candidate.Version != "" {
				parts = append(parts, "version="+candidate.Version)
			}
			if candidate.File != "" {
				parts = append(parts, "source="+candidate.File)
			}
			if candidate.Field != "" {
				parts = append(parts, "field="+candidate.Field)
			}
			fmt.Fprintf(out, "  - %s\n", strings.Join(parts, " "))
		}
	}
}
