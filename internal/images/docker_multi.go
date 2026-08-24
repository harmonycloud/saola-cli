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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func exportWithDockerMulti(ctx context.Context, runner Runner, images []ResolvedImage, output string, opts ExportOptions) error {
	platforms, err := requiredDockerMultiPlatforms(opts)
	if err != nil {
		return err
	}
	selectedRemote, err := validateRemotePlatforms(ctx, runner, images, platforms, opts)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "saola-images-docker-multi-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	layoutDir := filepath.Join(tmpDir, "layout")
	progressOut := progressOutput(opts)
	for i, item := range images {
		printImageProgress(progressOut, i, len(images), "exporting multi-platform", item.Image)
		selectedPlatforms := selectedRemote[item.Image]
		staged := make([]stagedOCIPlatform, 0, len(selectedPlatforms))
		for platformIndex, platform := range selectedPlatforms {
			platformName := ociPlatformString(platform)
			platformArgs, platformErr := skopeoPlatformArgs(platformName)
			if platformErr != nil {
				return platformErr
			}
			stageRef := fmt.Sprintf("saola-stage-%d-%d", i, platformIndex)
			fmt.Fprintf(progressOut, "Exporting %s for %s\n", platformName, item.Image)
			args := append([]string{"copy"}, platformArgs...)
			if opts.Insecure {
				args = append(args, "--src-tls-verify=false")
			}
			args = append(args, "docker://"+item.Image, "oci:"+layoutDir+":"+stageRef)
			if err = runStreaming(ctx, runner, progressOut, progressOut, toolSkopeo, args...); err != nil {
				return fmt.Errorf("export %s image %s with skopeo: %w", platformName, item.Image, err)
			}
			staged = append(staged, stagedOCIPlatform{Ref: stageRef, Platform: platform})
		}
		if err = composeOCIMultiPlatformImage(layoutDir, item.Image, staged); err != nil {
			return fmt.Errorf("compose multi-platform image %s: %w", item.Image, err)
		}
		images[i].Platforms = ociPlatformNames(selectedPlatforms)
		printImageProgress(progressOut, i+1, len(images), "exported multi-platform", item.Image)
	}

	fmt.Fprintf(progressOut, "Packing Docker multi-platform archive -> %s\n", output)
	return tarDirectory(output, layoutDir)
}

func requiredDockerMultiPlatforms(opts ExportOptions) ([]platformSpec, error) {
	platforms, err := normalizeRequiredPlatforms(opts.Platforms)
	if err != nil {
		return nil, err
	}
	if len(platforms) < 2 {
		return nil, fmt.Errorf("--format=%s requires at least two --platforms values; use --format=%s for a single platform", ExportFormatDockerMulti, ExportFormatDocker)
	}
	if opts.Platform != "" && opts.Platform != "all" {
		return nil, fmt.Errorf("--format=%s uses --platforms instead of --platform", ExportFormatDockerMulti)
	}
	return platforms, nil
}

func validateRemotePlatforms(ctx context.Context, runner Runner, images []ResolvedImage, required []platformSpec, opts ExportOptions) (map[string][]ociPlatform, error) {
	selected := make(map[string][]ociPlatform, len(images))
	for _, item := range images {
		if len(item.Platforms) == 0 {
			continue
		}
		available, err := ociPlatformsFromNames(item.Platforms)
		if err != nil {
			return nil, fmt.Errorf("parse resolved platforms for %s: %w", item.Image, err)
		}
		if missing := missingRequiredPlatforms(required, available); len(missing) > 0 {
			return nil, fmt.Errorf("image %s is missing required platforms %s (available: %s)", item.Image, strings.Join(missing, ", "), formatOCIPlatforms(available))
		}
		selected[item.Image] = selectRequiredOCIPlatforms(required, available)
	}
	inspectRunner, ok := runner.(outputRunner)
	if !ok {
		for _, item := range images {
			if len(selected[item.Image]) == 0 {
				selected[item.Image] = requestedOCIPlatforms(required)
			}
		}
		return selected, nil
	}
	for _, item := range images {
		if len(selected[item.Image]) > 0 {
			continue
		}
		available, err := inspectRemotePlatforms(ctx, inspectRunner, item.Image, opts)
		if err != nil {
			return nil, fmt.Errorf("inspect platforms for %s: %w", item.Image, err)
		}
		if missing := missingRequiredPlatforms(required, available); len(missing) > 0 {
			return nil, fmt.Errorf("image %s is missing required platforms %s (available: %s)", item.Image, strings.Join(missing, ", "), formatOCIPlatforms(available))
		}
		selected[item.Image] = selectRequiredOCIPlatforms(required, available)
	}
	return selected, nil
}

func inspectRemotePlatforms(ctx context.Context, runner outputRunner, image string, opts ExportOptions) ([]ociPlatform, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultProbeTimeout
	}
	args := []string{"inspect", "--raw"}
	if opts.Insecure {
		args = append(args, "--tls-verify=false")
	}
	args = append(args, "docker://"+image)
	data, err := runOutputWithTimeout(ctx, runner, timeout, toolSkopeo, args...)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Manifests []ociDescriptor `json:"manifests"`
	}
	if err = json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse raw manifest: %w", err)
	}
	if len(raw.Manifests) > 0 {
		platforms := make([]ociPlatform, 0, len(raw.Manifests))
		for _, descriptor := range raw.Manifests {
			if validOCIPlatform(descriptor.Platform) {
				platforms = append(platforms, *descriptor.Platform)
			}
		}
		return uniqueOCIPlatforms(platforms), nil
	}

	args = []string{"inspect", "--config"}
	if opts.Insecure {
		args = append(args, "--tls-verify=false")
	}
	args = append(args, "docker://"+image)
	data, err = runOutputWithTimeout(ctx, runner, timeout, toolSkopeo, args...)
	if err != nil {
		return nil, err
	}
	var config ociImageConfig
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse image config: %w", err)
	}
	if config.OS == "" || config.Architecture == "" {
		return nil, fmt.Errorf("image config does not declare os and architecture")
	}
	return []ociPlatform{{OS: config.OS, Architecture: config.Architecture, Variant: config.Variant}}, nil
}
