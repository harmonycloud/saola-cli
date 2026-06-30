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
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ImportOptions controls importing an exported image archive into a target registry.
//
// ImportOptions 控制把导出的镜像归档导入（推送）到目标仓库。
type ImportOptions struct {
	Archive     string
	Repository  string
	LockFile    string
	Creds       string
	Platform    string
	MultiArch   bool
	Insecure    bool
	DryRun      bool
	ProgressOut io.Writer
	Runner      Runner
}

// ImportTarget describes one image to push: its tag inside the OCI archive and the destination reference.
//
// ImportTarget 描述一个待推送镜像：归档内的 OCI tag 以及目标镜像引用。
type ImportTarget struct {
	Name        string
	OCITag      string
	Destination string
}

// ImportResult summarizes one import run.
//
// ImportResult 汇总一次导入的结果。
type ImportResult struct {
	Archive    string
	Repository string
	Targets    []ImportTarget
	DryRun     bool
}

// ImportPackage pushes images from an exported OCI layout archive into the target repository.
//
// ImportPackage 把导出的 OCI layout 归档中的镜像推送到目标仓库。
func ImportPackage(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	if opts.Archive == "" {
		return nil, fmt.Errorf("archive path is required")
	}
	if opts.Repository == "" {
		return nil, fmt.Errorf("at least one --repository is required")
	}
	if _, err := os.Stat(opts.Archive); err != nil {
		return nil, fmt.Errorf("archive %s: %w", opts.Archive, err)
	}
	if opts.Platform == "" {
		opts.Platform = "all"
	}
	repo := strings.TrimRight(strings.TrimSpace(opts.Repository), "/")

	names, err := importImageNames(opts)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no images found in lock file or archive index")
	}

	targets := make([]ImportTarget, 0, len(names))
	for _, name := range names {
		targets = append(targets, ImportTarget{
			Name:        name,
			OCITag:      safeOCITag(name),
			Destination: repo + "/" + name,
		})
	}

	result := &ImportResult{
		Archive:    opts.Archive,
		Repository: repo,
		Targets:    targets,
		DryRun:     opts.DryRun,
	}
	if opts.DryRun {
		return result, nil
	}

	// Only skopeo can read an OCI layout and push it to a registry directly;
	// docker/nerdctl cannot `load` an OCI layout archive, so fail clearly instead.
	//
	// 只有 skopeo 能读取 OCI layout 并直接推送到仓库；docker/nerdctl 无法 load
	// OCI layout 归档，因此这里直接给出清晰报错，而不是悄悄走一条不可用的回退。
	runner := defaultRunner(opts.Runner)
	if _, err = runner.LookPath(toolSkopeo); err != nil {
		return nil, fmt.Errorf("skopeo is required to import an OCI layout archive (docker/nerdctl cannot load it directly); please install skopeo")
	}

	tmpDir, err := os.MkdirTemp("", "saola-images-import-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	progressOut := importProgress(opts)
	fmt.Fprintf(progressOut, "Extracting image archive <- %s\n", opts.Archive)
	if err = untarArchive(opts.Archive, tmpDir); err != nil {
		return nil, fmt.Errorf("extract archive: %w", err)
	}
	ociDir := filepath.Join(tmpDir, "images")
	if _, err = os.Stat(filepath.Join(ociDir, "index.json")); err != nil {
		return nil, fmt.Errorf("archive does not contain an OCI layout under images/ (is this a saola images export archive?): %w", err)
	}

	if err = importWithSkopeo(ctx, runner, ociDir, targets, opts, progressOut); err != nil {
		return nil, err
	}
	return result, nil
}

func importWithSkopeo(ctx context.Context, runner Runner, ociDir string, targets []ImportTarget, opts ImportOptions, progressOut io.Writer) error {
	for i, t := range targets {
		printImageProgress(progressOut, i, len(targets), "importing", t.Destination)
		args := []string{"copy"}
		if opts.MultiArch && opts.Platform == "all" {
			args = append(args, "--all")
		} else if opts.Platform != "" && opts.Platform != "all" {
			platformArgs, perr := skopeoPlatformArgs(opts.Platform)
			if perr != nil {
				return perr
			}
			args = append(args, platformArgs...)
		}
		if opts.Insecure {
			args = append(args, "--dest-tls-verify=false")
		}
		if opts.Creds != "" {
			args = append(args, "--dest-creds", opts.Creds)
		}
		args = append(args, "oci:"+ociDir+":"+t.OCITag, "docker://"+t.Destination)
		if err := runStreaming(ctx, runner, progressOut, progressOut, toolSkopeo, args...); err != nil {
			return fmt.Errorf("import %s with skopeo: %w", t.Destination, err)
		}
		printImageProgress(progressOut, i+1, len(targets), "imported", t.Destination)
	}
	return nil
}

// importImageNames resolves the image name list from the export lock file.
//
// importImageNames 从 export 生成的 lock 文件解析镜像名清单。
func importImageNames(opts ImportOptions) ([]string, error) {
	lockPath := opts.LockFile
	if lockPath == "" {
		lockPath = opts.Archive + ".lock.json"
	}
	if data, err := os.ReadFile(lockPath); err == nil {
		var lock LockFile
		if err = json.Unmarshal(data, &lock); err != nil {
			return nil, fmt.Errorf("parse lock file %s: %w", lockPath, err)
		}
		return dedupeNames(lockImageNames(lock)), nil
	} else {
		return nil, fmt.Errorf("read lock file %s: %w", lockPath, err)
	}
}

func lockImageNames(lock LockFile) []string {
	names := make([]string, 0, len(lock.Images))
	for _, img := range lock.Images {
		if img.Name != "" {
			names = append(names, img.Name)
		}
	}
	return names
}

func dedupeNames(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, n := range in {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// untarArchive extracts a tar archive into dest, guarding against path traversal.
//
// untarArchive 把 tar 归档解包到 dest，并防止路径穿越。
func untarArchive(archive, dest string) (err error) {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	cleanDest := filepath.Clean(dest)
	tr := tar.NewReader(f)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(cleanDest, filepath.Clean("/"+header.Name))
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr) //nolint:gosec // archive is produced by saola export
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	return nil
}

func importProgress(opts ImportOptions) io.Writer {
	if opts.ProgressOut != nil {
		return opts.ProgressOut
	}
	return io.Discard
}
