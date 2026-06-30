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

	imageexport "github.com/harmonycloud/saola-cli/internal/images"

	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// ImportOptions holds parameters for "images import".
//
// ImportOptions 保存 "images import" 命令参数。
type ImportOptions struct {
	Config     *config.Config
	Archive    string
	Repository string
	LockFile   string
	Creds      string
	Platform   string
	MultiArch  bool
	Insecure   bool
	DryRun     bool
	Runner     imageexport.Runner
	Out        io.Writer
	ErrOut     io.Writer
}

// NewCmdImport returns the "images import" command.
//
// NewCmdImport 返回 images import 子命令。
func NewCmdImport(cfg *config.Config) *cobra.Command {
	o := &ImportOptions{
		Config:    cfg,
		Platform:  "all",
		MultiArch: true,
		Out:       os.Stdout,
		ErrOut:    os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "import <archive>",
		Short: lang.T("把导出的镜像归档推送到指定仓库", "Push an exported image archive to a target repository"),
		Long: lang.T(
			`把 saola images export 生成的 OCI layout 归档导入（推送）到指定镜像仓库。镜像名清单来自归档旁的 .lock.json 或显式 --lock-file，缺失时会报错以避免丢失原始 name:tag。`,
			`Import (push) an OCI layout archive produced by "saola images export" into a target image repository. The image name list is read from the sibling .lock.json or explicit --lock-file; missing lock files fail to preserve original name:tag references.`,
		),
		Example: `  saola images import milvus-images.tar -r 10.10.101.172:443/middleware
  saola images import milvus-images.tar -r 10.10.101.172:443/middleware --creds "$REGISTRY_CREDS" --insecure
  saola images import milvus-images.tar -r 10.10.101.172:443/middleware --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Archive = args[0]
			o.Out = cmd.OutOrStdout()
			o.ErrOut = cmd.ErrOrStderr()
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Repository, "repository", "r", "", lang.T("目标镜像仓库，例如 10.10.101.172:443/middleware", "Target image repository, for example 10.10.101.172:443/middleware"))
	cmd.Flags().StringVar(&o.LockFile, "lock-file", "", lang.T("镜像锁定清单路径（默认：<archive>.lock.json）", "Image lock file path (default: <archive>.lock.json)"))
	cmd.Flags().StringVar(&o.Creds, "creds", "", lang.T("目标仓库凭据，格式 user:password", "Target registry credentials in user:password form"))
	cmd.Flags().StringVar(&o.Platform, "platform", "all", lang.T("导入平台，例如 linux/amd64；all 表示保留多架构", "Import platform, for example linux/amd64; all keeps multi-arch images"))
	cmd.Flags().BoolVar(&o.MultiArch, "multi-arch", true, lang.T("使用 skopeo 推送时保留多架构清单", "Keep multi-arch manifests when pushing with skopeo"))
	cmd.Flags().BoolVar(&o.Insecure, "insecure", false, lang.T("跳过目标仓库 TLS 校验", "Skip target registry TLS verification"))
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, lang.T("仅打印将要推送的镜像，不执行", "Print images to push without executing"))
	return cmd
}

// Run executes image import.
//
// Run 执行镜像导入。
func (o *ImportOptions) Run(ctx context.Context) error {
	out := o.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := o.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}

	result, err := imageexport.ImportPackage(ctx, imageexport.ImportOptions{
		Archive:     o.Archive,
		Repository:  o.Repository,
		LockFile:    o.LockFile,
		Creds:       o.Creds,
		Platform:    o.Platform,
		MultiArch:   o.MultiArch,
		Insecure:    o.Insecure,
		DryRun:      o.DryRun,
		ProgressOut: errOut,
		Runner:      o.Runner,
	})
	if err != nil {
		return err
	}

	if result.DryRun {
		fmt.Fprintf(out, "Would import %d images from %s -> %s\n", len(result.Targets), result.Archive, result.Repository)
		for _, t := range result.Targets {
			fmt.Fprintf(out, "  - %s (oci tag: %s)\n", t.Destination, t.OCITag)
		}
		return nil
	}

	fmt.Fprintf(out, "Imported %d images from %s -> %s\n", len(result.Targets), result.Archive, result.Repository)
	return nil
}
