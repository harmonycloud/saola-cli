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
	"testing"
	"time"
)

func TestDiscoverPackageImages_ExpandsMultipleRepositories(t *testing.T) {
	t.Parallel()
	dir := makeImagePackage(t)

	meta, groups, err := DiscoverPackageImages(dir, []string{
		"repo-a.example:5000/middleware,repo-b.example:5000/middleware",
	})
	if err != nil {
		t.Fatalf("DiscoverPackageImages: %v", err)
	}
	if meta.Name != "Redis" || meta.Version != "2.20.1-1.0.1" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}

	assertCandidateImages(t, groups, "redis-cli-port:v8.2.6-1.0.0-redis", []string{
		"repo-a.example:5000/middleware/redis-cli-port:v8.2.6-1.0.0-redis",
		"repo-b.example:5000/middleware/redis-cli-port:v8.2.6-1.0.0-redis",
	})
	assertCandidateImages(t, groups, "redis-init:v1.7.4", []string{
		"repo-a.example:5000/middleware/redis-init:v1.7.4",
		"repo-b.example:5000/middleware/redis-init:v1.7.4",
	})
	assertCandidateImages(t, groups, "redis-operator:v2.20.1", []string{
		"repo-a.example:5000/middleware/redis-operator:v2.20.1",
		"repo-b.example:5000/middleware/redis-operator:v2.20.1",
	})
	assertCandidateImages(t, groups, "redis-sidecar:v1.2.3", []string{
		"repo-a.example:5000/middleware/redis-sidecar:v1.2.3",
		"repo-b.example:5000/middleware/redis-sidecar:v1.2.3",
	})
	assertCandidateImages(t, groups, "redis-dashboard:v8.2.6", []string{
		"repo-a.example:5000/middleware/redis-dashboard:v8.2.6",
		"repo-b.example:5000/middleware/redis-dashboard:v8.2.6",
	})
	assertCandidateImages(t, groups, "redis-exporter:v8.2.6", []string{
		"repo-a.example:5000/middleware/redis-exporter:v8.2.6",
		"repo-b.example:5000/middleware/redis-exporter:v8.2.6",
	})
	assertCandidateImages(t, groups, "redis-audit:v1.0.0", []string{
		"repo-a.example:5000/middleware/redis-audit:v1.0.0",
		"repo-b.example:5000/middleware/redis-audit:v1.0.0",
	})
}

func TestExportPackage_DryRunBuildsLock(t *testing.T) {
	t.Parallel()
	dir := makeImagePackage(t)

	result, err := ExportPackage(context.Background(), ExportOptions{
		PkgDir:       dir,
		Repositories: []string{"repo-a.example:5000/middleware", "repo-b.example:5000/middleware"},
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("ExportPackage dry-run: %v", err)
	}
	if len(result.Groups) == 0 {
		t.Fatal("expected discovered image groups")
	}
	if len(result.Lock.Repositories) != 2 {
		t.Fatalf("expected two lock repositories, got %#v", result.Lock.Repositories)
	}
	if len(result.Resolved) != 0 {
		t.Fatalf("dry-run should not resolve images, got %#v", result.Resolved)
	}
}

func TestDiscoverPackageImages_ReferencedMultiDocumentConfiguration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "metadata.yaml"), `name: MultiDoc
version: 1.0.0
app:
  version:
    - "1.0.0"
`)
	writeFile(t, filepath.Join(dir, "baselines", "cluster.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareBaseline
metadata:
  name: multidoc-cluster
spec:
  necessary:
    repository: ""
    version: '{"type":"version","default":"1.0.0"}'
  configurations:
    - name: second-doc
      values:
        repository: "{{ .Necessary.repository }}"
        tag: "{{ .Necessary.version }}"
`)
	writeFile(t, filepath.Join(dir, "configurations", "multi.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareConfiguration
metadata:
  name: first-doc
spec:
  template: |-
    apiVersion: apps/v1
    kind: Deployment
    spec:
      template:
        spec:
          containers:
            - name: first
              image: "{{ .Values.repository }}/first:{{ .Values.tag }}"
---
apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareConfiguration
metadata:
  name: second-doc
spec:
  template: |-
    apiVersion: apps/v1
    kind: Deployment
    spec:
      template:
        spec:
          containers:
            - name: second
              image: "{{ .Values.repository }}/second:{{ .Values.tag }}"
`)

	_, groups, err := DiscoverPackageImages(dir, []string{"repo.example/middleware"})
	if err != nil {
		t.Fatalf("DiscoverPackageImages: %v", err)
	}
	assertCandidateImages(t, groups, "second:1.0.0", []string{"repo.example/middleware/second:1.0.0"})
	assertNoImageGroup(t, groups, "first:1.0.0")
}

func TestResolveImages_UsesFirstExistingRepository(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{
		tools: map[string]bool{toolSkopeo: true},
		existing: map[string]bool{
			"repo-b.example:5000/middleware/redis-init:v1.7.4": true,
		},
	}
	groups := []ImageGroup{{
		Name: "redis-init:v1.7.4",
		Candidates: []ImageCandidate{
			{Image: "repo-a.example:5000/middleware/redis-init:v1.7.4", Repository: "repo-a.example:5000/middleware"},
			{Image: "repo-b.example:5000/middleware/redis-init:v1.7.4", Repository: "repo-b.example:5000/middleware"},
		},
	}}

	resolved, missing, err := resolveImages(context.Background(), runner, groups, ExportOptions{Timeout: time.Second})
	if err != nil {
		t.Fatalf("resolveImages: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing images, got %#v", missing)
	}
	if len(resolved) != 1 || resolved[0].Image != "repo-b.example:5000/middleware/redis-init:v1.7.4" {
		t.Fatalf("unexpected resolved images: %#v", resolved)
	}
	if len(runner.probed) != 2 || runner.probed[0] != "repo-a.example:5000/middleware/redis-init:v1.7.4" || runner.probed[1] != "repo-b.example:5000/middleware/redis-init:v1.7.4" {
		t.Fatalf("unexpected probe order: %#v", runner.probed)
	}
}

func TestResolveImages_RegistryTLSErrorKeepsCandidateContext(t *testing.T) {
	t.Parallel()

	image := "10.10.101.172:443/middleware/kubectl:v1.30.14"
	runner := &fakeRunner{
		tools: map[string]bool{toolSkopeo: true},
		inspectErrors: map[string]error{
			image: errors.New("x509: certificate signed by unknown authority"),
		},
	}
	groups := []ImageGroup{{
		Name: "kubectl:v1.30.14",
		Candidates: []ImageCandidate{{
			Image:      image,
			Repository: "10.10.101.172:443/middleware",
			File:       "chart/templates/upgrade-crds.yaml",
			Field:      "spec.template.spec.containers[0].image",
		}},
	}}

	resolved, missing, err := resolveImages(context.Background(), runner, groups, ExportOptions{Timeout: time.Second})
	if err != nil {
		t.Fatalf("resolveImages: %v", err)
	}
	if len(resolved) != 0 {
		t.Fatalf("expected no resolved images, got %#v", resolved)
	}
	if len(missing) != 1 || len(missing[0].ProbeErrors) != 1 {
		t.Fatalf("expected missing image with probe error, got %#v", missing)
	}

	msg := formatMissingImagesError(missing)
	for _, want := range []string{
		"image=kubectl:v1.30.14",
		"candidate=10.10.101.172:443/middleware/kubectl:v1.30.14",
		"file=chart/templates/upgrade-crds.yaml",
		"field=spec.template.spec.containers[0].image",
		"reason=RegistryTLS",
		"x509: certificate signed by unknown authority",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected missing image error to contain %q, got %q", want, msg)
		}
	}
}

func TestResolveImages_DockerMultiFallsBackToCandidateWithRequiredPlatforms(t *testing.T) {
	t.Parallel()
	first := "repo-a.example/middleware/redis:v1.0.0"
	second := "repo-b.example/middleware/redis:v1.0.0"
	base := &fakeRunner{tools: map[string]bool{toolSkopeo: true}}
	runner := &fakeOutputRunner{
		fakeRunner: base,
		outputHook: func(_ string, args []string) ([]byte, error) {
			image := strings.TrimPrefix(args[len(args)-1], "docker://")
			switch image {
			case first:
				if containsArg(args, "--config") {
					return []byte(`{"os":"linux","architecture":"amd64"}`), nil
				}
				return []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`), nil
			case second:
				return []byte(`{"schemaVersion":2,"manifests":[{"platform":{"os":"linux","architecture":"amd64"}},{"platform":{"os":"linux","architecture":"arm64","variant":"v8"}}]}`), nil
			default:
				return nil, errors.New("unexpected image " + image)
			}
		},
	}
	groups := []ImageGroup{{
		Name: "redis:v1.0.0",
		Candidates: []ImageCandidate{
			{Image: first, Repository: "repo-a.example/middleware"},
			{Image: second, Repository: "repo-b.example/middleware"},
		},
	}}

	resolved, missing, err := resolveImages(context.Background(), runner, groups, ExportOptions{
		Format:    ExportFormatDockerMulti,
		Platforms: []string{"linux/amd64", "linux/arm64"},
		Timeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("resolveImages: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing images, got %#v", missing)
	}
	if len(resolved) != 1 || resolved[0].Image != second {
		t.Fatalf("expected multi-platform fallback candidate, got %#v", resolved)
	}
	if got := strings.Join(resolved[0].Platforms, ","); got != "linux/amd64,linux/arm64/v8" {
		t.Fatalf("unexpected resolved platforms %q", got)
	}
	if len(base.runs) != 0 {
		t.Fatalf("multi-platform resolution should use manifest output without duplicate inspect runs, got %#v", base.runs)
	}
}

func TestExportWithSkopeo_UsesPlatformOverrideParts(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolSkopeo: true}}
	output := filepath.Join(t.TempDir(), "images.tar")

	err := exportWithSkopeo(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, output, ExportOptions{
		Platform:  "linux/amd64",
		Insecure:  true,
		Timeout:   time.Second,
		MultiArch: true,
	})
	if err != nil {
		t.Fatalf("exportWithSkopeo: %v", err)
	}
	if len(runner.runs) != 1 {
		t.Fatalf("expected one skopeo copy run, got %#v", runner.runs)
	}
	got := strings.Join(runner.runs[0].args, " ")
	for _, want := range []string{
		"--override-os linux",
		"--override-arch amd64",
		"--src-tls-verify=false",
		"docker://repo.example/middleware/redis:v1.0.0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected args to contain %q, got %q", want, got)
		}
	}
	if strings.Contains(got, "--override-platform") {
		t.Fatalf("skopeo does not support --override-platform, got %q", got)
	}
}

func TestExportImages_FormatDockerUsesDockerArchive(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolSkopeo: true, toolDocker: true}}

	err := exportImages(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, filepath.Join(t.TempDir(), "images.tar"), ExportOptions{
		Format:   ExportFormatDocker,
		Platform: "linux/amd64",
	})
	if err != nil {
		t.Fatalf("exportImages: %v", err)
	}
	if len(runner.runs) != 2 {
		t.Fatalf("expected docker pull and save, got %#v", runner.runs)
	}
	if runner.runs[0].name != toolDocker || runner.runs[0].args[0] != "pull" {
		t.Fatalf("expected docker pull, got %#v", runner.runs[0])
	}
	if runner.runs[1].name != toolDocker || runner.runs[1].args[0] != "save" {
		t.Fatalf("expected docker save, got %#v", runner.runs[1])
	}
}

func TestExportImages_FormatDockerInsecureUsesSkopeoPullAndDockerSave(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolSkopeo: true, toolDocker: true}}

	err := exportImages(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, filepath.Join(t.TempDir(), "images.tar"), ExportOptions{
		Format:   ExportFormatDocker,
		Platform: "linux/amd64",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("exportImages: %v", err)
	}
	if len(runner.runs) != 2 {
		t.Fatalf("expected skopeo copy and docker save, got %#v", runner.runs)
	}
	if runner.runs[0].name != toolSkopeo {
		t.Fatalf("expected skopeo copy, got %#v", runner.runs[0])
	}
	got := strings.Join(runner.runs[0].args, " ")
	for _, want := range []string{
		"copy",
		"--override-os linux",
		"--override-arch amd64",
		"--src-tls-verify=false",
		"docker://repo.example/middleware/redis:v1.0.0",
		"docker-daemon:repo.example/middleware/redis:v1.0.0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected args to contain %q, got %q", want, got)
		}
	}
	if runner.runs[1].name != toolDocker || runner.runs[1].args[0] != "save" {
		t.Fatalf("expected docker save, got %#v", runner.runs[1])
	}
}

func TestExportImages_FormatDockerInsecureRequiresSkopeo(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolDocker: true}}

	err := exportImages(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, filepath.Join(t.TempDir(), "images.tar"), ExportOptions{
		Format:   ExportFormatDocker,
		Insecure: true,
	})
	if err == nil || !strings.Contains(err.Error(), "requires skopeo") {
		t.Fatalf("expected skopeo-required error, got %v", err)
	}
	if len(runner.runs) != 0 {
		t.Fatalf("expected no commands after dependency check failed, got %#v", runner.runs)
	}
}

func TestExportImages_FormatDockerMultiBuildsLoadableOCIRoot(t *testing.T) {
	t.Parallel()
	ref := "repo.example:5000/middleware/redis:v1.0.0"
	runner := &fakeRunner{
		tools: map[string]bool{toolSkopeo: true},
		runHook: func(name string, args []string) error {
			if name != toolSkopeo || len(args) < 4 || args[0] != "copy" {
				return errors.New("unexpected command")
			}
			got := strings.Join(args, " ")
			for _, want := range []string{"--override-os linux", "--src-tls-verify=false", "docker://" + ref} {
				if !strings.Contains(got, want) {
					return errors.New("missing argument " + want)
				}
			}
			if strings.Contains(got, "--all") {
				return errors.New("docker-multi should only copy requested platforms")
			}
			destination := args[len(args)-1]
			layoutAndRef := strings.TrimPrefix(destination, "oci:")
			parts := strings.SplitN(layoutAndRef, ":", 2)
			if len(parts) != 2 || !strings.HasPrefix(parts[1], "saola-stage-") {
				return errors.New("unexpected OCI destination " + destination)
			}
			platform := ociPlatform{
				OS:           argValue(args, "--override-os"),
				Architecture: argValue(args, "--override-arch"),
				Variant:      argValue(args, "--override-variant"),
			}
			return writeFakeOCIStage(parts[0], parts[1], platform)
		},
	}
	images := []ResolvedImage{{Name: "redis:v1.0.0", Image: ref}}
	output := filepath.Join(t.TempDir(), "images.tar")

	err := exportImages(context.Background(), runner, images, output, ExportOptions{
		Format:    ExportFormatDockerMulti,
		Platform:  "all",
		Platforms: []string{"linux/amd64", "linux/arm64"},
		Insecure:  true,
	})
	if err != nil {
		t.Fatalf("exportImages: %v", err)
	}
	if got := strings.Join(images[0].Platforms, ","); got != "linux/amd64,linux/arm64" {
		t.Fatalf("unexpected selected platforms %q", got)
	}
	if len(runner.runs) != 2 {
		t.Fatalf("expected one skopeo copy per required platform, got %#v", runner.runs)
	}

	archive, err := os.Open(output)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer archive.Close()
	entries := map[string]bool{}
	reader := tar.NewReader(archive)
	for {
		header, readErr := reader.Next()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			t.Fatalf("read archive: %v", readErr)
		}
		entries[header.Name] = true
	}
	if !entries["oci-layout"] || !entries["index.json"] {
		t.Fatalf("expected root OCI layout files, got %#v", entries)
	}
	if entries["images/oci-layout"] || entries["images/index.json"] {
		t.Fatalf("docker-multi archive must keep the OCI layout at archive root, got %#v", entries)
	}
}

func TestExportImages_FormatDockerMultiRequiresSkopeo(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolDocker: true}}

	err := exportImages(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, filepath.Join(t.TempDir(), "images.tar"), ExportOptions{Format: ExportFormatDockerMulti})
	if err == nil || !strings.Contains(err.Error(), "requires skopeo") {
		t.Fatalf("expected skopeo-required error, got %v", err)
	}
}

func TestExportPackage_DockerMultiWritesPlatformsToLock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "metadata.yaml"), `name: BusyboxMulti
version: 1.0.0
app:
  version:
    - latest
`)
	writeFile(t, filepath.Join(dir, "baselines", "default.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareBaseline
metadata:
  name: busybox-multi
spec:
  operator:
    image: busybox:latest
`)
	ref := "repo.example/middleware/busybox:latest"
	runner := &fakeRunner{
		tools:    map[string]bool{toolSkopeo: true},
		existing: map[string]bool{ref: true},
		runHook: func(name string, args []string) error {
			if name != toolSkopeo || len(args) == 0 || args[0] != "copy" {
				return nil
			}
			destination := strings.TrimPrefix(args[len(args)-1], "oci:")
			parts := strings.SplitN(destination, ":", 2)
			if len(parts) != 2 {
				return errors.New("unexpected OCI destination")
			}
			return writeFakeOCIStage(parts[0], parts[1], ociPlatform{
				OS:           argValue(args, "--override-os"),
				Architecture: argValue(args, "--override-arch"),
				Variant:      argValue(args, "--override-variant"),
			})
		},
	}
	output := filepath.Join(t.TempDir(), "images.tar")

	result, err := ExportPackage(context.Background(), ExportOptions{
		PkgDir:       dir,
		Output:       output,
		Repositories: []string{"repo.example/middleware"},
		Format:       ExportFormatDockerMulti,
		Platform:     "all",
		Platforms:    []string{"linux/amd64", "linux/arm64"},
		Runner:       runner,
	})
	if err != nil {
		t.Fatalf("ExportPackage: %v", err)
	}
	if len(result.Resolved) != 1 || strings.Join(result.Resolved[0].Platforms, ",") != "linux/amd64,linux/arm64" {
		t.Fatalf("unexpected resolved platforms %#v", result.Resolved)
	}
	if len(result.Lock.Images) != 1 || strings.Join(result.Lock.Images[0].Platforms, ",") != "linux/amd64,linux/arm64" {
		t.Fatalf("unexpected lock platforms %#v", result.Lock.Images)
	}
	data, err := os.ReadFile(output + ".lock.json")
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	if !strings.Contains(string(data), `"platforms": [`) || !strings.Contains(string(data), `"linux/arm64"`) {
		t.Fatalf("lock file does not contain platform metadata: %s", data)
	}
}

func TestExportPackage_DockerMultiRejectsSinglePlatformBeforeDiscovery(t *testing.T) {
	t.Parallel()
	_, err := ExportPackage(context.Background(), ExportOptions{
		PkgDir:       filepath.Join(t.TempDir(), "missing-package"),
		Repositories: []string{"repo.example/middleware"},
		Format:       ExportFormatDockerMulti,
		Platforms:    []string{"linux/amd64"},
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least two --platforms") {
		t.Fatalf("expected early platform validation error, got %v", err)
	}
}

func TestValidateRemotePlatformsRejectsSingleArchitectureBeforeCopy(t *testing.T) {
	t.Parallel()
	base := &fakeRunner{tools: map[string]bool{toolSkopeo: true}}
	runner := &fakeOutputRunner{
		fakeRunner: base,
		outputHook: func(name string, args []string) ([]byte, error) {
			if name != toolSkopeo || !strings.Contains(strings.Join(args, " "), "--tls-verify=false") {
				return nil, errors.New("unexpected inspect command")
			}
			if containsArg(args, "--raw") {
				return []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`), nil
			}
			if containsArg(args, "--config") {
				return []byte(`{"os":"linux","architecture":"amd64"}`), nil
			}
			return nil, errors.New("unexpected inspect mode")
		},
	}
	required, err := normalizeRequiredPlatforms([]string{"linux/amd64", "linux/arm64"})
	if err != nil {
		t.Fatalf("normalizeRequiredPlatforms: %v", err)
	}

	_, err = validateRemotePlatforms(context.Background(), runner, []ResolvedImage{{Image: "repo.example/middleware/redis:v1.0.0"}}, required, ExportOptions{
		Insecure: true,
		Timeout:  time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "missing required platforms linux/arm64") {
		t.Fatalf("expected missing arm64 error, got %v", err)
	}
	if len(base.runs) != 0 {
		t.Fatalf("preflight validation must fail before copy commands, got %#v", base.runs)
	}
}

func TestComposeOCIMultiPlatformImagePreservesOtherImages(t *testing.T) {
	t.Parallel()
	layoutDir := t.TempDir()
	refs := []string{
		"repo.example/middleware/redis:v1.0.0",
		"repo.example/middleware/exporter:v1.0.0",
	}
	for imageIndex, ref := range refs {
		staged := []stagedOCIPlatform{
			{Ref: fmt.Sprintf("stage-%d-amd64", imageIndex), Platform: ociPlatform{OS: "linux", Architecture: "amd64"}},
			{Ref: fmt.Sprintf("stage-%d-arm64", imageIndex), Platform: ociPlatform{OS: "linux", Architecture: "arm64"}},
		}
		for _, item := range staged {
			if err := writeFakeOCIStage(layoutDir, item.Ref, item.Platform); err != nil {
				t.Fatalf("writeFakeOCIStage: %v", err)
			}
		}
		if err := composeOCIMultiPlatformImage(layoutDir, ref, staged); err != nil {
			t.Fatalf("composeOCIMultiPlatformImage: %v", err)
		}
	}
	data, err := os.ReadFile(filepath.Join(layoutDir, "index.json"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	var index ociIndexDocument
	if err = json.Unmarshal(data, &index); err != nil {
		t.Fatalf("parse index: %v", err)
	}
	if len(index.Manifests) != len(refs) {
		t.Fatalf("expected %d image references, got %#v", len(refs), index.Manifests)
	}
	gotRefs := map[string]bool{}
	for _, descriptor := range index.Manifests {
		gotRefs[descriptor.Annotations[ociRefNameAnnotation]] = true
	}
	for _, ref := range refs {
		if !gotRefs[ref] {
			t.Fatalf("missing composed image reference %s in %#v", ref, gotRefs)
		}
	}
}

func TestExportImages_FormatDockerFallsBackToNerdctl(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolNerdctl: true}}

	err := exportImages(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, filepath.Join(t.TempDir(), "images.tar"), ExportOptions{Format: ExportFormatDocker})
	if err != nil {
		t.Fatalf("exportImages: %v", err)
	}
	if len(runner.runs) != 2 {
		t.Fatalf("expected nerdctl pull and save, got %#v", runner.runs)
	}
	if runner.runs[0].name != toolNerdctl || runner.runs[0].args[0] != "pull" {
		t.Fatalf("expected nerdctl pull, got %#v", runner.runs[0])
	}
	if runner.runs[1].name != toolNerdctl || runner.runs[1].args[0] != "save" {
		t.Fatalf("expected nerdctl save, got %#v", runner.runs[1])
	}
}

func TestExportImages_FormatSkopeoRequiresSkopeo(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolDocker: true}}

	err := exportImages(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, filepath.Join(t.TempDir(), "images.tar"), ExportOptions{Format: ExportFormatSkopeo})
	if err == nil || !strings.Contains(err.Error(), "skopeo") {
		t.Fatalf("expected skopeo-required error, got %v", err)
	}
}

func TestExportImages_RejectsUnknownFormat(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{tools: map[string]bool{toolSkopeo: true}}

	err := exportImages(context.Background(), runner, []ResolvedImage{{
		Name:  "redis:v1.0.0",
		Image: "repo.example/middleware/redis:v1.0.0",
	}}, filepath.Join(t.TempDir(), "images.tar"), ExportOptions{Format: "oci"})
	if err == nil || !strings.Contains(err.Error(), "unsupported image export format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestSkopeoPlatformArgsRejectsInvalidPlatform(t *testing.T) {
	t.Parallel()
	if _, err := skopeoPlatformArgs("linux"); err == nil {
		t.Fatal("expected invalid platform error")
	}
}

func makeImagePackage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "metadata.yaml"), `name: Redis
version: 2.20.1-1.0.1
app:
  version:
    - "8.2.6"
`)
	writeFile(t, filepath.Join(dir, "baselines", "cluster.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareBaseline
metadata:
  name: redis-cluster
spec:
  necessary:
    repository: ""
    version: '{"type":"version","default":"8.2.6"}'
  globe:
    repository: "old.example:443"
    project: "old-project"
  parameters:
    pod:
      middlewareImage: '"{{- $version := .Necessary.version -}}{{- $image := dict -}}{{- $_ := set $image "redisImageTag_8_2_6" "v8.2.6-1.0.0-redis" -}}{{- $versionUnderscore := replace "." "_" $version -}}{{- $imageTagKey := printf "redisImageTag_%s" $versionUnderscore -}}{{- printf "redis-cli-port:%s" (index $image $imageTagKey) -}}"'
      initImage: "redis-init:v1.7.4"
  operator:
    image: '{{ if ne .Globe.repository "" }}{{ .Globe.repository }}/{{ end }}{{ .Globe.project }}/redis-operator:v2.20.1'
  sidecar:
    repository: "{{ .Necessary.repository }}/redis-sidecar"
    tag: "v1.2.3"
  configurations:
    - name: redis-dashboard
      values:
        repository: "{{ .Necessary.repository }}"
        version: "{{ .Necessary.version }}"
    - name: redis-exporter
      values:
        repository: "{{ .Necessary.repository }}"
        tag: "{{ .Necessary.version }}"
    - name: redis-audit
      values:
        repository: "{{ .Necessary.repository }}"
        tag: "v1.0.0"
        labels:
          app: redis-audit
`)
	writeFile(t, filepath.Join(dir, "configurations", "redis-dashboard.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareConfiguration
metadata:
  name: redis-dashboard
spec:
  template:
    spec:
      containers:
        - name: dashboard
          image: "{{ .Values.repository }}/redis-dashboard:v{{ .Values.version }}"
`)
	writeFile(t, filepath.Join(dir, "configurations", "redis-exporter.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareConfiguration
metadata:
  name: redis-exporter
spec:
  template: |-
    {{- $repo := tpl (toString (.Values.repository | default .Necessary.repository)) . }}
    {{- $tag := tpl (toString (.Values.tag | default .Necessary.version)) . }}
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: redis-exporter
    spec:
      template:
        spec:
          containers:
            - name: exporter
              image: "{{ $repo }}/redis-exporter:v{{ $tag }}"
`)
	writeFile(t, filepath.Join(dir, "configurations", "redis-audit.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareConfiguration
metadata:
  name: redis-audit
spec:
  template: |-
    {{- $_ := required "repository is required" .Values.repository }}
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: redis-audit
      labels:
        {{- toYaml .Values.labels | nindent 4 }}
    spec:
      template:
        spec:
          containers:
            - name: audit
              image: "{{ .Values.repository }}/redis-audit:{{ .Values.tag }}"
`)
	return dir
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func assertCandidateImages(t *testing.T, groups []ImageGroup, name string, want []string) {
	t.Helper()
	for _, group := range groups {
		if group.Name != name {
			continue
		}
		got := make([]string, 0, len(group.Candidates))
		for _, candidate := range group.Candidates {
			got = append(got, candidate.Image)
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("candidates for %s mismatch:\nwant: %#v\n got: %#v", name, want, got)
		}
		return
	}
	t.Fatalf("group %q not found in %#v", name, groups)
}

func assertNoImageGroup(t *testing.T, groups []ImageGroup, name string) {
	t.Helper()
	for _, group := range groups {
		if group.Name == name {
			t.Fatalf("group %q should not be present: %#v", name, group)
		}
	}
}

type fakeRunner struct {
	tools         map[string]bool
	existing      map[string]bool
	inspectErrors map[string]error
	runHook       func(name string, args []string) error
	probed        []string
	runs          []fakeRun
}

type fakeRun struct {
	name string
	args []string
}

type fakeOutputRunner struct {
	*fakeRunner
	outputHook func(name string, args []string) ([]byte, error)
}

func (r *fakeOutputRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if r.outputHook == nil {
		return nil, errors.New("output not configured")
	}
	return r.outputHook(name, args)
}

func (r *fakeRunner) LookPath(file string) (string, error) {
	if r.tools[file] {
		return file, nil
	}
	return "", errors.New("not found")
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	r.runs = append(r.runs, fakeRun{name: name, args: append([]string(nil), args...)})
	if r.runHook != nil {
		if err := r.runHook(name, args); err != nil {
			return err
		}
	}
	if name != toolSkopeo || len(args) < 1 || args[0] != "inspect" {
		return nil
	}
	image := strings.TrimPrefix(args[len(args)-1], "docker://")
	r.probed = append(r.probed, image)
	if err := r.inspectErrors[image]; err != nil {
		return err
	}
	if r.existing[image] {
		return nil
	}
	return errors.New("manifest unknown")
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func argValue(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

func writeFakeOCIStage(layoutDir string, ref string, platform ociPlatform) error {
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(layoutDir, "oci-layout"), []byte(`{"imageLayoutVersion":"1.0.0"}`), 0o644); err != nil {
		return err
	}
	configData, err := json.Marshal(ociImageConfig{OS: platform.OS, Architecture: platform.Architecture, Variant: platform.Variant})
	if err != nil {
		return err
	}
	configDigest, configSize, err := writeOCIBlob(layoutDir, configData)
	if err != nil {
		return err
	}
	manifestData, err := json.Marshal(ociManifestDocument{
		SchemaVersion: 2,
		MediaType:     ociManifestMediaType,
		Config: ociDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    configDigest,
			Size:      configSize,
		},
	})
	if err != nil {
		return err
	}
	manifestDigest, manifestSize, err := writeOCIBlob(layoutDir, manifestData)
	if err != nil {
		return err
	}

	index := ociIndexDocument{SchemaVersion: 2, MediaType: ociIndexMediaType}
	indexPath := filepath.Join(layoutDir, "index.json")
	if data, readErr := os.ReadFile(indexPath); readErr == nil {
		if err = json.Unmarshal(data, &index); err != nil {
			return err
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	index.Manifests = append(index.Manifests, ociDescriptor{
		MediaType:   ociManifestMediaType,
		Digest:      manifestDigest,
		Size:        manifestSize,
		Annotations: map[string]string{ociRefNameAnnotation: ref},
	})
	indexData, err := json.Marshal(index)
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath, indexData, 0o644)
}
