package pkgcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gitea.com/middleware-management/saola-cli/internal/config"
	"gitea.com/middleware-management/saola-cli/internal/lang"
	"gitea.com/middleware-management/saola-cli/internal/packager"
	"github.com/spf13/cobra"
)

// BuildOptions holds parameters for "package build".
// BuildOptions 保存 "package build" 命令的参数。
type BuildOptions struct {
	Config *config.Config
	PkgDir string
	Output string
}

// NewCmdBuild returns the "package build" command.
//
// 返回 package build 子命令。
func NewCmdBuild(cfg *config.Config) *cobra.Command {
	o := &BuildOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "build <pkg-dir>",
		Short: lang.T("从本地目录构建包归档文件", "Build a package archive from a local directory"),
		Long: lang.T(
			`将本地目录打包为 zstd 压缩的 TAR 文件，不执行安装。
适用于 CI 流水线或离线分发场景。`,
			`Pack the local directory into a zstd-compressed TAR file without installing it.
Useful for CI pipelines or offline distribution.`,
		),
		Example: `  saola package build .
  saola package build ./my-redis --output ./dist/redis-v1.pkg`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.PkgDir = args[0]
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "", lang.T("输出文件路径（默认：当前目录下的 <name>-<version>.pkg）", "Output file path (default: <name>-<version>.pkg in current dir)"))
	return cmd
}

func (o *BuildOptions) Run() error {
	fmt.Fprintf(os.Stdout, "Packing directory %s ...\n", o.PkgDir)
	data, meta, err := packager.PackDir(o.PkgDir)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}

	outPath := o.Output
	if outPath == "" {
		outPath = fmt.Sprintf("%s-%s.pkg", meta.Name, meta.Version)
	}
	// Ensure the output directory exists.
	if dir := filepath.Dir(outPath); dir != "." {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	if err = os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Built %s@%s -> %s (%d bytes)\n", meta.Name, meta.Version, outPath, len(data))
	return nil
}
