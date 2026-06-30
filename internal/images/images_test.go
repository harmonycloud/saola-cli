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
	"errors"
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
	probed        []string
	runs          []fakeRun
}

type fakeRun struct {
	name string
	args []string
}

func (r *fakeRunner) LookPath(file string) (string, error) {
	if r.tools[file] {
		return file, nil
	}
	return "", errors.New("not found")
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	r.runs = append(r.runs, fakeRun{name: name, args: append([]string(nil), args...)})
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
