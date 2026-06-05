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
	"time"
)

const (
	toolSkopeo  = "skopeo"
	toolDocker  = "docker"
	toolNerdctl = "nerdctl"
)

// ExportResult contains the resolved images and generated lock file.
//
// ExportResult 保存解析后的镜像和生成的锁定清单。
type ExportResult struct {
	Metadata PackageMetadata
	Groups   []ImageGroup
	Resolved []ResolvedImage
	Missing  []MissingImage
	Lock     LockFile
	Output   string
	LockPath string
}

// ExportPackage resolves package images and optionally exports them.
//
// ExportPackage 解析包镜像，并按需导出镜像归档。
func ExportPackage(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	if opts.PkgDir == "" {
		return nil, fmt.Errorf("pkg dir is required")
	}
	opts.Repositories = normalizeRepositories(opts.Repositories)
	if len(opts.Repositories) == 0 {
		return nil, fmt.Errorf("at least one --repository is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Platform == "" {
		opts.Platform = "all"
	}

	meta, groups, err := DiscoverPackageImages(opts.PkgDir, opts.Repositories)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("no images found in %s", opts.PkgDir)
	}

	result := &ExportResult{
		Metadata: meta,
		Groups:   groups,
	}
	if opts.DryRun {
		result.Lock = buildLock(meta, opts.Repositories, nil, nil)
		return result, nil
	}

	runner := defaultRunner(opts.Runner)
	resolved, missing, err := resolveImages(ctx, runner, groups, opts)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 && !opts.SkipMissing {
		return nil, fmt.Errorf("%d images could not be resolved; use --skip-missing to export available images", len(missing))
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("no images resolved")
	}

	output := opts.Output
	if output == "" {
		output = fmt.Sprintf("%s-%s-images.tar", meta.Name, meta.Version)
	}
	lockPath := opts.LockFile
	if lockPath == "" {
		lockPath = output + ".lock.json"
	}
	lock := buildLock(meta, opts.Repositories, resolved, missing)

	if err = exportImages(ctx, runner, resolved, output, opts); err != nil {
		return nil, err
	}
	if err = writeLockFile(lockPath, lock); err != nil {
		return nil, err
	}

	result.Resolved = resolved
	result.Missing = missing
	result.Lock = lock
	result.Output = output
	result.LockPath = lockPath
	return result, nil
}

func resolveImages(ctx context.Context, runner Runner, groups []ImageGroup, opts ExportOptions) ([]ResolvedImage, []MissingImage, error) {
	if _, err := runner.LookPath(toolSkopeo); err != nil {
		if _, dockerErr := runner.LookPath(toolDocker); dockerErr != nil {
			if _, nerdErr := runner.LookPath(toolNerdctl); nerdErr != nil {
				return nil, nil, fmt.Errorf("no image inspection tool found; install skopeo, docker, or nerdctl")
			}
		}
	}

	var resolved []ResolvedImage
	var missing []MissingImage
	for _, group := range groups {
		var hit *ImageCandidate
		inspected := map[string]bool{}
		for _, candidate := range group.Candidates {
			if inspected[candidate.Image] {
				continue
			}
			inspected[candidate.Image] = true
			ok, err := inspectImage(ctx, runner, candidate.Image, opts)
			if err != nil {
				continue
			}
			if ok {
				copyCandidate := candidate
				hit = &copyCandidate
				break
			}
		}
		if hit == nil {
			missing = append(missing, MissingImage{
				Name:       group.Name,
				Candidates: group.Candidates,
			})
			continue
		}
		resolved = append(resolved, ResolvedImage{
			Name:       group.Name,
			Image:      hit.Image,
			Repository: hit.Repository,
			File:       hit.File,
			Field:      hit.Field,
			Version:    hit.Version,
			Candidates: group.Candidates,
		})
	}
	return resolved, missing, nil
}

func inspectImage(ctx context.Context, runner Runner, image string, opts ExportOptions) (bool, error) {
	runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if _, err := runner.LookPath(toolSkopeo); err == nil {
		args := []string{"inspect", "--raw"}
		if opts.Insecure {
			args = append(args, "--tls-verify=false")
		}
		args = append(args, "docker://"+image)
		if err = runner.Run(runCtx, toolSkopeo, args...); err != nil {
			return false, err
		}
		return true, nil
	}

	if _, err := runner.LookPath(toolDocker); err == nil {
		if err = runner.Run(runCtx, toolDocker, "manifest", "inspect", image); err != nil {
			return false, err
		}
		return true, nil
	}

	if _, err := runner.LookPath(toolNerdctl); err == nil {
		if err = runner.Run(runCtx, toolNerdctl, "image", "inspect", image); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, fmt.Errorf("no image inspection tool found")
}

func exportImages(ctx context.Context, runner Runner, images []ResolvedImage, output string, opts ExportOptions) error {
	if _, err := runner.LookPath(toolSkopeo); err == nil {
		return exportWithSkopeo(ctx, runner, images, output, opts)
	}
	if _, err := runner.LookPath(toolDocker); err == nil {
		return exportWithDocker(ctx, runner, images, output, opts)
	}
	if _, err := runner.LookPath(toolNerdctl); err == nil {
		return exportWithNerdctl(ctx, runner, images, output, opts)
	}
	return fmt.Errorf("no image export tool found; install skopeo, docker, or nerdctl")
}

func exportWithSkopeo(ctx context.Context, runner Runner, images []ResolvedImage, output string, opts ExportOptions) error {
	tmpDir, err := os.MkdirTemp("", "saola-images-oci-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	ociDir := filepath.Join(tmpDir, "images")
	for _, item := range images {
		runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		args := []string{"copy"}
		if opts.MultiArch && opts.Platform == "all" {
			args = append(args, "--all")
		} else if opts.Platform != "" && opts.Platform != "all" {
			args = append(args, "--override-platform", opts.Platform)
		}
		if opts.Insecure {
			args = append(args, "--src-tls-verify=false")
		}
		args = append(args, "docker://"+item.Image, "oci:"+ociDir+":"+safeOCITag(item.Name))
		err = runner.Run(runCtx, toolSkopeo, args...)
		cancel()
		if err != nil {
			return fmt.Errorf("export %s with skopeo: %w", item.Image, err)
		}
	}
	return tarDirectory(output, tmpDir)
}

func exportWithDocker(ctx context.Context, runner Runner, images []ResolvedImage, output string, opts ExportOptions) error {
	imageNames := make([]string, 0, len(images))
	for _, item := range images {
		runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		args := []string{"pull"}
		if opts.Platform != "" && opts.Platform != "all" {
			args = append(args, "--platform", opts.Platform)
		}
		args = append(args, item.Image)
		err := runner.Run(runCtx, toolDocker, args...)
		cancel()
		if err != nil {
			return fmt.Errorf("pull %s with docker: %w", item.Image, err)
		}
		imageNames = append(imageNames, item.Image)
	}
	args := append([]string{"save", "-o", output}, imageNames...)
	return runner.Run(ctx, toolDocker, args...)
}

func exportWithNerdctl(ctx context.Context, runner Runner, images []ResolvedImage, output string, opts ExportOptions) error {
	imageNames := make([]string, 0, len(images))
	for _, item := range images {
		runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		args := []string{"pull"}
		if opts.Platform != "" && opts.Platform != "all" {
			args = append(args, "--platform", opts.Platform)
		}
		args = append(args, item.Image)
		err := runner.Run(runCtx, toolNerdctl, args...)
		cancel()
		if err != nil {
			return fmt.Errorf("pull %s with nerdctl: %w", item.Image, err)
		}
		imageNames = append(imageNames, item.Image)
	}
	args := append([]string{"save", "-o", output}, imageNames...)
	return runner.Run(ctx, toolNerdctl, args...)
}

func buildLock(meta PackageMetadata, repos []string, resolved []ResolvedImage, missing []MissingImage) LockFile {
	return LockFile{
		Package:      meta,
		GeneratedAt:  time.Now().UTC(),
		Repositories: repos,
		Images:       resolved,
		Missing:      missing,
	}
}

func writeLockFile(path string, lock LockFile) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func tarDirectory(output, dir string) error {
	if parent := filepath.Dir(output); parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	tw := tar.NewWriter(out)
	defer tw.Close()

	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err = tw.WriteHeader(header); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, in)
		closeErr := in.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}

func safeOCITag(name string) string {
	out := strings.Builder{}
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
			out.WriteRune(ch)
		case ch >= 'A' && ch <= 'Z':
			out.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			out.WriteRune(ch)
		case ch == '.', ch == '_', ch == '-':
			out.WriteRune(ch)
		default:
			out.WriteRune('-')
		}
	}
	return strings.Trim(out.String(), "-")
}
