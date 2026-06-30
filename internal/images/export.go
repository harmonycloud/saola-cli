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
	"errors"
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
		opts.Timeout = DefaultProbeTimeout
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
		return nil, fmt.Errorf("%s; use --skip-missing to export available images", formatMissingImagesError(missing))
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
		var probeErrors []ProbeError
		for _, candidate := range group.Candidates {
			if inspected[candidate.Image] {
				continue
			}
			inspected[candidate.Image] = true
			ok, err := inspectImage(ctx, runner, candidate.Image, opts)
			if err != nil {
				probeErrors = append(probeErrors, ProbeError{
					Image:      candidate.Image,
					Repository: candidate.Repository,
					File:       candidate.File,
					Field:      candidate.Field,
					Reason:     classifyProbeError(err),
					Message:    err.Error(),
				})
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
				Name:        group.Name,
				Candidates:  group.Candidates,
				ProbeErrors: probeErrors,
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

func formatMissingImagesError(missing []MissingImage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d images could not be resolved", len(missing))
	for _, item := range missing {
		fmt.Fprintf(&b, "; image=%s", item.Name)
		for _, probe := range item.ProbeErrors {
			fmt.Fprintf(&b, " candidate=%s", probe.Image)
			if probe.File != "" {
				fmt.Fprintf(&b, " file=%s", probe.File)
			}
			if probe.Field != "" {
				fmt.Fprintf(&b, " field=%s", probe.Field)
			}
			if probe.Reason != "" {
				fmt.Fprintf(&b, " reason=%s", probe.Reason)
			}
			if probe.Message != "" {
				fmt.Fprintf(&b, " message=%s", probe.Message)
			}
		}
	}
	return b.String()
}

func classifyProbeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "x509") || strings.Contains(msg, "certificate signed by unknown authority") || strings.Contains(msg, "tls"):
		return "RegistryTLS"
	case strings.Contains(msg, "unauthorized") || strings.Contains(msg, "authentication required") || strings.Contains(msg, "denied"):
		return "RegistryAuth"
	case strings.Contains(msg, "manifest unknown") || strings.Contains(msg, "name unknown") || strings.Contains(msg, "not found"):
		return "ManifestNotFound"
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
		return "Timeout"
	default:
		return "InspectFailed"
	}
}

func inspectImage(ctx context.Context, runner Runner, image string, opts ExportOptions) (bool, error) {
	if _, err := runner.LookPath(toolSkopeo); err == nil {
		args := []string{"inspect", "--raw"}
		if opts.Insecure {
			args = append(args, "--tls-verify=false")
		}
		args = append(args, "docker://"+image)
		if err = runWithTimeout(ctx, runner, opts.Timeout, toolSkopeo, args...); err != nil {
			return false, err
		}
		return true, nil
	}

	if _, err := runner.LookPath(toolDocker); err == nil {
		if err = runWithTimeout(ctx, runner, opts.Timeout, toolDocker, "manifest", "inspect", image); err != nil {
			return false, err
		}
		return true, nil
	}

	if _, err := runner.LookPath(toolNerdctl); err == nil {
		if err = runWithTimeout(ctx, runner, opts.Timeout, toolNerdctl, "image", "inspect", image); err != nil {
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
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	ociDir := filepath.Join(tmpDir, "images")
	progressOut := progressOutput(opts)
	for i, item := range images {
		printImageProgress(progressOut, i, len(images), "exporting", item.Image)
		args := []string{"copy"}
		if opts.MultiArch && opts.Platform == "all" {
			args = append(args, "--all")
		} else if opts.Platform != "" && opts.Platform != "all" {
			platformArgs, platformErr := skopeoPlatformArgs(opts.Platform)
			if platformErr != nil {
				return platformErr
			}
			args = append(args, platformArgs...)
		}
		if opts.Insecure {
			args = append(args, "--src-tls-verify=false")
		}
		args = append(args, "docker://"+item.Image, "oci:"+ociDir+":"+safeOCITag(item.Name))
		if err = runStreaming(ctx, runner, progressOut, progressOut, toolSkopeo, args...); err != nil {
			return fmt.Errorf("export %s with skopeo: %w", item.Image, err)
		}
		printImageProgress(progressOut, i+1, len(images), "exported", item.Image)
	}
	fmt.Fprintf(progressOut, "Packing image archive -> %s\n", output)
	return tarDirectory(output, tmpDir)
}

func exportWithDocker(ctx context.Context, runner Runner, images []ResolvedImage, output string, opts ExportOptions) error {
	imageNames := make([]string, 0, len(images))
	progressOut := progressOutput(opts)
	for i, item := range images {
		printImageProgress(progressOut, i, len(images), "pulling", item.Image)
		args := []string{"pull"}
		if opts.Platform != "" && opts.Platform != "all" {
			args = append(args, "--platform", opts.Platform)
		}
		args = append(args, item.Image)
		if err := runStreaming(ctx, runner, progressOut, progressOut, toolDocker, args...); err != nil {
			return fmt.Errorf("pull %s with docker: %w", item.Image, err)
		}
		printImageProgress(progressOut, i+1, len(images), "pulled", item.Image)
		imageNames = append(imageNames, item.Image)
	}
	fmt.Fprintf(progressOut, "Packing image archive -> %s\n", output)
	args := append([]string{"save", "-o", output}, imageNames...)
	return runStreaming(ctx, runner, progressOut, progressOut, toolDocker, args...)
}

func exportWithNerdctl(ctx context.Context, runner Runner, images []ResolvedImage, output string, opts ExportOptions) error {
	imageNames := make([]string, 0, len(images))
	progressOut := progressOutput(opts)
	for i, item := range images {
		printImageProgress(progressOut, i, len(images), "pulling", item.Image)
		args := []string{"pull"}
		if opts.Platform != "" && opts.Platform != "all" {
			args = append(args, "--platform", opts.Platform)
		}
		args = append(args, item.Image)
		if err := runStreaming(ctx, runner, progressOut, progressOut, toolNerdctl, args...); err != nil {
			return fmt.Errorf("pull %s with nerdctl: %w", item.Image, err)
		}
		printImageProgress(progressOut, i+1, len(images), "pulled", item.Image)
		imageNames = append(imageNames, item.Image)
	}
	fmt.Fprintf(progressOut, "Packing image archive -> %s\n", output)
	args := append([]string{"save", "-o", output}, imageNames...)
	return runStreaming(ctx, runner, progressOut, progressOut, toolNerdctl, args...)
}

func runWithTimeout(ctx context.Context, runner Runner, timeout time.Duration, name string, args ...string) error {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := runner.Run(runCtx, name, args...)
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timed out after %s", timeout)
	}
	return err
}

type streamingRunner interface {
	RunStreaming(ctx context.Context, stdout io.Writer, stderr io.Writer, name string, args ...string) error
}

func runStreaming(ctx context.Context, runner Runner, stdout io.Writer, stderr io.Writer, name string, args ...string) error {
	if stderr == nil {
		stderr = io.Discard
	}
	start := time.Now()
	done := make(chan error, 1)
	go func() {
		if stream, ok := runner.(streamingRunner); ok {
			done <- stream.RunStreaming(ctx, stdout, stderr, name, args...)
			return
		}
		done <- runner.Run(ctx, name, args...)
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			fmt.Fprintf(stderr, "Still running %s %s (%s elapsed)\n", name, commandAction(args), time.Since(start).Round(time.Second))
		}
	}
}

func progressOutput(opts ExportOptions) io.Writer {
	if opts.ProgressOut != nil {
		return opts.ProgressOut
	}
	return io.Discard
}

func commandAction(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func printImageProgress(out io.Writer, done int, total int, action string, image string) {
	if out == nil || total <= 0 {
		return
	}
	width := 20
	filled := done * width / total
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("=", filled) + strings.Repeat(".", width-filled)
	fmt.Fprintf(out, "[%s] %d/%d %s %s\n", bar, done, total, action, image)
}

func skopeoPlatformArgs(platform string) ([]string, error) {
	if platform == "" || platform == "all" {
		return nil, nil
	}
	parts := strings.Split(platform, "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" || (len(parts) == 3 && parts[2] == "") {
		return nil, fmt.Errorf("invalid platform %q, expected os/arch[/variant] or all", platform)
	}
	args := []string{"--override-os", parts[0], "--override-arch", parts[1]}
	if len(parts) == 3 {
		args = append(args, "--override-variant", parts[2])
	}
	return args, nil
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

func tarDirectory(output, dir string) (err error) {
	if parent := filepath.Dir(output); parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	tw := tar.NewWriter(out)
	defer func() {
		if closeErr := tw.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
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
	return err
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
