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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

func TestNewCmdExport_DryRun(t *testing.T) {
	t.Parallel()
	dir := makeCmdImagePackage(t)
	cmd := NewCmdExport(&config.Config{})
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{
		dir,
		"-r", "repo-a.example:5000/middleware,repo-b.example:5000/middleware",
		"--dry-run",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Found",
		"repo-a.example:5000/middleware/redis-cli-port:v8.2.6-1.0.0-redis",
		"repo-b.example:5000/middleware/redis-cli-port:v8.2.6-1.0.0-redis",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestNewCmdImages_HasExport(t *testing.T) {
	t.Parallel()
	cmd := NewCmdImages(&config.Config{})
	if _, _, err := cmd.Find([]string{"export"}); err != nil {
		t.Fatalf("expected export command: %v", err)
	}
}

func makeCmdImagePackage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeCmdFile(t, filepath.Join(dir, "metadata.yaml"), `name: Redis
version: 2.20.1-1.0.1
app:
  version:
    - "8.2.6"
`)
	writeCmdFile(t, filepath.Join(dir, "baseline.yaml"), `apiVersion: middleware.harmonycloud.cn/v1
kind: MiddlewareBaseline
spec:
  necessary:
    repository: ""
    version: '{"type":"version","default":"8.2.6"}'
  parameters:
    pod:
      middlewareImage: '"{{- $version := .Necessary.version -}}{{- $image := dict -}}{{- $_ := set $image "redisImageTag_8_2_6" "v8.2.6-1.0.0-redis" -}}{{- $versionUnderscore := replace "." "_" $version -}}{{- $imageTagKey := printf "redisImageTag_%s" $versionUnderscore -}}{{- printf "redis-cli-port:%s" (index $image $imageTagKey) -}}"'
`)
	return dir
}

func writeCmdFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
