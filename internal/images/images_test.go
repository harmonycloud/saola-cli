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

type fakeRunner struct {
	tools    map[string]bool
	existing map[string]bool
	probed   []string
}

func (r *fakeRunner) LookPath(file string) (string, error) {
	if r.tools[file] {
		return file, nil
	}
	return "", errors.New("not found")
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	if name != toolSkopeo || len(args) < 1 || args[0] != "inspect" {
		return nil
	}
	image := strings.TrimPrefix(args[len(args)-1], "docker://")
	r.probed = append(r.probed, image)
	if r.existing[image] {
		return nil
	}
	return errors.New("manifest unknown")
}
