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

package packagevalidator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validFiles(template string) map[string][]byte {
	return map[string][]byte{
		"metadata.yaml": []byte("name: testpkg\nversion: \"1.0.0\"\n"),
		"configurations/config.yaml": []byte(`apiVersion: middleware.cn/v1
kind: MiddlewareConfiguration
metadata:
  name: test-config
spec:
  template: |
` + indent(template, "    ")),
	}
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func TestValidateFilesValidConfigurationTemplate(t *testing.T) {
	files := validFiles(`apiVersion: v1
kind: ConfigMap
metadata:
  name: "{{ .Globe.Name }}"
  labels:
{{- toYaml .Globe.Labels | nindent 4 }}
data:
  ok: "true"
`)

	result, err := ValidateFiles(files)
	if err != nil {
		t.Fatalf("expected valid package files, got %v", err)
	}
	if result.Templates != 1 {
		t.Fatalf("expected 1 rendered template, got %d", result.Templates)
	}
}

func TestValidateFilesRejectsTemplateParseError(t *testing.T) {
	files := validFiles(`apiVersion: v1
kind: ConfigMap
metadata:
  name: "{{ .Globe.Name }}"
data:
  skills.list: |
  {{- .with .Values.skillsList }}
  {{- toYaml . | nindent 4 }}
  {{- end }}
`)

	_, err := ValidateFiles(files)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "unexpected {{end}}") {
		t.Fatalf("expected template parse error, got %v", err)
	}
}

func TestValidateFilesRejectsRenderedYAMLError(t *testing.T) {
	files := validFiles(`apiVersion: v1
kind: ConfigMap
metadata:
  name: "{{ .Globe.Name }}"
  labels: {{ .Globe.Labels }}
data:
  ok: "true"
`)

	_, err := ValidateFiles(files)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "metadata.labels must be a map") {
		t.Fatalf("expected rendered metadata.labels error, got %v", err)
	}
}

func TestValidateFilesRequiresMetadata(t *testing.T) {
	_, err := ValidateFiles(map[string][]byte{
		"configurations/config.yaml": []byte("apiVersion: v1\nkind: ConfigMap\n"),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "metadata.yaml is required") {
		t.Fatalf("expected metadata error, got %v", err)
	}
}

func TestValidateDirReadsPackageFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte("name: testpkg\nversion: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	configDir := filepath.Join(dir, "configurations")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configurations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), validFiles(`apiVersion: v1
kind: ConfigMap
metadata:
  name: "{{ .Globe.Name }}"
`)["configurations/config.yaml"], 0o644); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	if _, err := ValidateDir(dir); err != nil {
		t.Fatalf("expected directory to validate, got %v", err)
	}
}
