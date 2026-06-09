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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/packager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

// Result summarizes a package validation run.
//
// Result 汇总一次包校验结果。
type Result struct {
	Files     int `json:"files" yaml:"files"`
	Documents int `json:"documents" yaml:"documents"`
	Templates int `json:"templates" yaml:"templates"`
}

// Issue describes one validation failure.
//
// Issue 表示一个具体的校验失败。
type Issue struct {
	Path     string `json:"path" yaml:"path"`
	Document int    `json:"document,omitempty" yaml:"document,omitempty"`
	Message  string `json:"message" yaml:"message"`
}

// Error aggregates all validation issues.
//
// Error 聚合所有校验问题。
type Error struct {
	Issues []Issue `json:"issues" yaml:"issues"`
}

func (e *Error) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "package validation failed"
	}
	if len(e.Issues) == 1 {
		issue := e.Issues[0]
		return fmt.Sprintf("package validation failed: %s: %s", issue.location(), issue.Message)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "package validation failed with %d issues:", len(e.Issues))
	for _, issue := range e.Issues {
		fmt.Fprintf(&b, "\n  - %s: %s", issue.location(), issue.Message)
	}
	return b.String()
}

func (i Issue) location() string {
	if i.Document > 0 {
		return fmt.Sprintf("%s#doc%d", i.Path, i.Document)
	}
	return i.Path
}

type document struct {
	path  string
	index int
	data  []byte
	meta  metav1.TypeMeta
}

type configurationTemplate struct {
	path     string
	document int
	name     string
	template string
}

// ValidateDir validates a local middleware package directory.
//
// ValidateDir 校验本地中间件包目录。
func ValidateDir(dir string) (*Result, error) {
	files, err := readPackageFiles(dir)
	if err != nil {
		return nil, err
	}
	return ValidateFiles(files)
}

// ValidateFiles validates package files keyed by their package-relative path.
//
// ValidateFiles 校验以包内相对路径索引的文件内容。
func ValidateFiles(files map[string][]byte) (*Result, error) {
	validator := &validator{
		files:          files,
		result:         &Result{},
		valuesByConfig: map[string]map[string]any{},
	}
	validator.validateMetadata()
	validator.validateYAMLFiles()
	validator.validateConfigurationTemplates()
	if len(validator.issues) > 0 {
		return validator.result, &Error{Issues: validator.issues}
	}
	return validator.result, nil
}

type validator struct {
	files          map[string][]byte
	result         *Result
	documents      []document
	configurations []configurationTemplate
	valuesByConfig map[string]map[string]any
	issues         []Issue
}

func readPackageFiles(dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("stat package dir: %w", err)
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", rel, err)
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk package dir: %w", err)
	}
	return files, nil
}

func (v *validator) validateMetadata() {
	data, ok := v.files["metadata.yaml"]
	if !ok {
		v.addIssue("metadata.yaml", 0, "metadata.yaml is required")
		return
	}
	var meta packager.Metadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		v.addIssue("metadata.yaml", 0, fmt.Sprintf("parse metadata.yaml: %v", err))
		return
	}
	if err := meta.Validate(); err != nil {
		v.addIssue("metadata.yaml", 0, err.Error())
	}
}

func (v *validator) validateYAMLFiles() {
	paths := make([]string, 0, len(v.files))
	for path := range v.files {
		if isYAMLFile(path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		v.result.Files++
		v.validateYAMLFile(path, v.files[path])
	}
}

func (v *validator) validateYAMLFile(path string, data []byte) {
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for docIndex := 1; ; docIndex++ {
		docData, err := reader.Read()
		if err == io.EOF {
			return
		}
		if err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("read YAML document: %v", err))
			return
		}
		if len(bytes.TrimSpace(docData)) == 0 {
			continue
		}
		v.result.Documents++
		v.validateYAMLDocument(path, docIndex, docData)
	}
}

func (v *validator) validateYAMLDocument(path string, docIndex int, data []byte) {
	var meta metav1.TypeMeta
	if err := utilyaml.Unmarshal(data, &meta); err != nil {
		v.addIssue(path, docIndex, fmt.Sprintf("parse YAML: %v", err))
		return
	}
	doc := document{path: path, index: docIndex, data: data, meta: meta}
	v.documents = append(v.documents, doc)

	switch meta.Kind {
	case "MiddlewareBaseline":
		var obj zeusv1.MiddlewareBaseline
		if err := utilyaml.Unmarshal(data, &obj); err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("decode MiddlewareBaseline: %v", err))
			return
		}
		v.collectConfigurationValues(obj.Spec.Configurations)
	case "MiddlewareOperatorBaseline":
		var obj zeusv1.MiddlewareOperatorBaseline
		if err := utilyaml.Unmarshal(data, &obj); err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("decode MiddlewareOperatorBaseline: %v", err))
			return
		}
		v.collectConfigurationValues(obj.Spec.Configurations)
	case "MiddlewareActionBaseline":
		var obj zeusv1.MiddlewareActionBaseline
		if err := utilyaml.Unmarshal(data, &obj); err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("decode MiddlewareActionBaseline: %v", err))
		}
	case "MiddlewareConfiguration":
		var obj zeusv1.MiddlewareConfiguration
		if err := utilyaml.Unmarshal(data, &obj); err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("decode MiddlewareConfiguration: %v", err))
			return
		}
		v.configurations = append(v.configurations, configurationTemplate{
			path:     path,
			document: docIndex,
			name:     obj.Name,
			template: obj.Spec.Template,
		})
	}
}

func (v *validator) collectConfigurationValues(configs []zeusv1.Configuration) {
	for _, cfg := range configs {
		if cfg.Name == "" || len(cfg.Values.Raw) == 0 {
			continue
		}
		values := map[string]any{}
		if err := json.Unmarshal(cfg.Values.Raw, &values); err != nil {
			v.addIssue("configuration:"+cfg.Name, 0, fmt.Sprintf("decode configuration values: %v", err))
			continue
		}
		merged := v.valuesByConfig[cfg.Name]
		if merged == nil {
			merged = map[string]any{}
			v.valuesByConfig[cfg.Name] = merged
		}
		for key, value := range values {
			merged[key] = value
		}
	}
}

func (v *validator) validateConfigurationTemplates() {
	for _, cfg := range v.configurations {
		if strings.TrimSpace(cfg.template) == "" {
			v.addIssue(cfg.path, cfg.document, "MiddlewareConfiguration spec.template is required")
			continue
		}
		v.result.Templates++
		values := defaultTemplateValues()
		for key, value := range v.valuesByConfig[cfg.name] {
			values.Values[key] = value
		}
		rendered, err := renderTemplate(cfg.template, values)
		if err != nil {
			v.addIssue(cfg.path, cfg.document, fmt.Sprintf("render spec.template: %v", err))
			continue
		}
		v.validateRenderedYAML(cfg.path, cfg.document, rendered)
	}
}

func (v *validator) validateRenderedYAML(path string, docIndex int, rendered string) {
	if strings.Contains(rendered, "<no value>") {
		v.addIssue(path, docIndex, "rendered spec.template still contains <no value>")
		return
	}
	reader := utilyaml.NewYAMLReader(bufio.NewReader(strings.NewReader(rendered)))
	seen := false
	for renderedDocIndex := 1; ; renderedDocIndex++ {
		docData, err := reader.Read()
		if err == io.EOF {
			if !seen {
				v.addIssue(path, docIndex, "rendered spec.template is empty")
			}
			return
		}
		if err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("read rendered YAML document %d: %v", renderedDocIndex, err))
			return
		}
		if len(bytes.TrimSpace(docData)) == 0 {
			continue
		}
		seen = true
		var doc map[string]any
		if err := utilyaml.Unmarshal(docData, &doc); err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("parse rendered YAML document %d: %v", renderedDocIndex, err))
			return
		}
		if err := validateRenderedObject(doc); err != nil {
			v.addIssue(path, docIndex, fmt.Sprintf("validate rendered YAML document %d: %v", renderedDocIndex, err))
			return
		}
	}
}

func validateRenderedObject(doc map[string]any) error {
	metadata, ok := doc["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	for _, field := range []string{"labels", "annotations"} {
		value, exists := metadata[field]
		if !exists || value == nil {
			continue
		}
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("metadata.%s must be a map after rendering, got %T", field, value)
		}
	}
	return nil
}

func (v *validator) addIssue(path string, docIndex int, message string) {
	v.issues = append(v.issues, Issue{
		Path:     path,
		Document: docIndex,
		Message:  message,
	})
}

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
