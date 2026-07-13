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
	"strings"
	"testing"
)

func TestRenderImageTemplateExecutesInclude(t *testing.T) {
	t.Parallel()

	text := `{{- define "image.repository" -}}{{ .Values.repository }}/redis{{- end -}}
image: "{{ include "image.repository" . }}:v1.0.0"`
	got, err := renderImageTemplate(text, templateValues{
		Values: map[string]any{"repository": "repo.example/middleware"},
	})
	if err != nil {
		t.Fatalf("renderImageTemplate: %v", err)
	}
	if want := `image: "repo.example/middleware/redis:v1.0.0"`; !strings.Contains(got, want) {
		t.Fatalf("expected rendered template to contain %q, got %q", want, got)
	}
}
