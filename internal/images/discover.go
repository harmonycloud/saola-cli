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
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var embeddedImageLineRE = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_.-]*[Ii]mage[A-Za-z0-9_.-]*)\s*:\s*["']?([^"'\s]+)["']?`)

type packageYAMLFile struct {
	Rel      string
	Data     []byte
	Document int
}

// DiscoverPackageImages scans a package directory and returns logical image groups.
//
// DiscoverPackageImages 扫描包目录并返回逻辑镜像分组。
func DiscoverPackageImages(pkgDir string, repositories []string) (PackageMetadata, []ImageGroup, error) {
	meta, err := readMetadata(pkgDir)
	if err != nil {
		return PackageMetadata{}, nil, err
	}
	repositories = normalizeRepositories(repositories)
	versions := meta.App.Version
	if len(versions) == 0 {
		versions = []string{""}
	}

	files, err := yamlFiles(pkgDir)
	if err != nil {
		return PackageMetadata{}, nil, err
	}

	yamlFiles := make([]packageYAMLFile, 0, len(files))
	configurations := map[string]packageYAMLFile{}
	for _, file := range files {
		rel, relErr := filepath.Rel(pkgDir, file)
		if relErr != nil {
			return PackageMetadata{}, nil, relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "metadata.yaml" {
			continue
		}
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			return PackageMetadata{}, nil, fmt.Errorf("read %s: %w", rel, readErr)
		}
		docs, splitErr := yamlDocuments(rel, data)
		if splitErr != nil {
			return PackageMetadata{}, nil, splitErr
		}
		yamlFiles = append(yamlFiles, docs...)
		if strings.HasPrefix(rel, "configurations/") {
			for _, item := range docs {
				if name := metadataName(item.Data); name != "" {
					configurations[name] = item
				}
			}
		}
	}

	groups := make(map[string]*ImageGroup)
	for _, file := range yamlFiles {
		defaults, globe, parseErr := templateDefaults(file.Data)
		if parseErr != nil {
			// Keep scanning with safe defaults; some package files contain non-K8s YAML fragments.
			defaults = map[string]any{}
			globe = map[string]any{}
		}
		for _, version := range versions {
			for _, repo := range repositories {
				values := buildTemplateValues(meta, defaults, globe, version, repo)
				appendCandidates(groups, repositories, scanYAMLImages(file.Data, file.Rel, version, repo, repositories, values))
				appendCandidates(groups, repositories, scanReferencedConfigurationImages(file.Data, version, repo, repositories, values, configurations))
			}
		}
	}

	result := make([]ImageGroup, 0, len(groups))
	for _, group := range groups {
		sort.Slice(group.Candidates, func(i, j int) bool {
			left := group.Candidates[i]
			right := group.Candidates[j]
			if left.Repository != right.Repository {
				return repositoryOrder(left.Repository, repositories) < repositoryOrder(right.Repository, repositories)
			}
			if left.Version != right.Version {
				return left.Version < right.Version
			}
			if left.File != right.File {
				return left.File < right.File
			}
			return left.Field < right.Field
		})
		result = append(result, *group)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return meta, result, nil
}

func appendCandidates(groups map[string]*ImageGroup, repositories []string, candidates []ImageCandidate) {
	for _, candidate := range candidates {
		key := logicalImageName(candidate.Image, repositories)
		group := groups[key]
		if group == nil {
			group = &ImageGroup{Name: key}
			groups[key] = group
		}
		group.Candidates = appendUniqueCandidate(group.Candidates, candidate)
	}
}

func readMetadata(pkgDir string) (PackageMetadata, error) {
	data, err := os.ReadFile(filepath.Join(pkgDir, "metadata.yaml"))
	if err != nil {
		return PackageMetadata{}, fmt.Errorf("read metadata.yaml: %w", err)
	}
	var meta PackageMetadata
	if err = yaml.Unmarshal(data, &meta); err != nil {
		return PackageMetadata{}, fmt.Errorf("parse metadata.yaml: %w", err)
	}
	if meta.Name == "" || meta.Version == "" {
		return PackageMetadata{}, fmt.Errorf("metadata.yaml: name and version are required")
	}
	return meta, nil
}

func yamlFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Name() != "." && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".yaml", ".yml":
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func yamlDocuments(rel string, data []byte) ([]packageYAMLFile, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var docs []packageYAMLFile
	for docIndex := 1; ; docIndex++ {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse %s document %d: %w", rel, docIndex, err)
		}
		if emptyYAMLDocument(&node) {
			continue
		}
		docData, err := yaml.Marshal(&node)
		if err != nil {
			return nil, fmt.Errorf("encode %s document %d: %w", rel, docIndex, err)
		}
		docs = append(docs, packageYAMLFile{
			Rel:      rel,
			Data:     docData,
			Document: docIndex,
		})
	}
	return docs, nil
}

func emptyYAMLDocument(node *yaml.Node) bool {
	if node == nil || node.Kind == 0 {
		return true
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return true
		}
		return emptyYAMLDocument(node.Content[0])
	}
	return node.Kind == yaml.ScalarNode && node.Value == ""
}

func metadataName(data []byte) string {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err != nil {
			break
		}
		metadata := mappingChild(&node, "metadata")
		name := mappingChild(metadata, "name")
		if name != nil && name.Kind == yaml.ScalarNode {
			return name.Value
		}
	}
	return ""
}

type configurationRef struct {
	Name   string
	Values map[string]any
}

func scanReferencedConfigurationImages(baselineData []byte, version, repo string, repositories []string, values templateValues, configurations map[string]packageYAMLFile) []ImageCandidate {
	var candidates []ImageCandidate
	for _, ref := range configurationRefs(baselineData, values) {
		config, ok := configurations[ref.Name]
		if !ok {
			continue
		}
		configValues := values
		configValues.Values = ref.Values
		candidates = append(candidates, scanYAMLImages(config.Data, config.Rel, version, repo, repositories, configValues)...)
	}
	return candidates
}

func configurationRefs(data []byte, values templateValues) []configurationRef {
	var refs []configurationRef
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err != nil {
			break
		}
		spec := mappingChild(&node, "spec")
		configs := mappingChild(spec, "configurations")
		if configs == nil || configs.Kind != yaml.SequenceNode {
			continue
		}
		for _, item := range configs.Content {
			name := mappingScalar(item, "name")
			if name == "" {
				continue
			}
			ref := configurationRef{
				Name:   name,
				Values: map[string]any{},
			}
			if valuesNode := mappingChild(item, "values"); valuesNode != nil {
				if rendered, ok := renderValuesNode(valuesNode, values).(map[string]any); ok {
					ref.Values = rendered
				}
			}
			refs = append(refs, ref)
		}
	}
	return refs
}

func renderValuesNode(node *yaml.Node, values templateValues) any {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil
		}
		return renderValuesNode(node.Content[0], values)
	case yaml.MappingNode:
		out := map[string]any{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			out[node.Content[i].Value] = renderValuesNode(node.Content[i+1], values)
		}
		return out
	case yaml.SequenceNode:
		out := make([]any, 0, len(node.Content))
		for _, item := range node.Content {
			out = append(out, renderValuesNode(item, values))
		}
		return out
	case yaml.ScalarNode:
		rendered, err := renderImageTemplate(node.Value, values)
		if err != nil {
			return node.Value
		}
		return rendered
	default:
		return nil
	}
}

func scanYAMLImages(data []byte, file, version, repo string, repositories []string, values templateValues) []ImageCandidate {
	var candidates []ImageCandidate
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err != nil {
			break
		}
		candidates = append(candidates, scanNodeImages(&node, file, "", version, repo, repositories, values)...)
	}
	return candidates
}

func scanNodeImages(node *yaml.Node, file, path, version, repo string, repositories []string, values templateValues) []ImageCandidate {
	if node == nil {
		return nil
	}
	var candidates []ImageCandidate
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			candidates = append(candidates, scanNodeImages(child, file, path, version, repo, repositories, values)...)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			value := node.Content[i+1]
			nextPath := joinPath(path, key)
			if isImageField(key) {
				candidates = append(candidates, candidatesFromScalar(value, file, nextPath, version, repo, repositories, values)...)
			}
			if key == "repository" {
				candidates = append(candidates, repositoryTagCandidate(node, file, path, version, repo, repositories, values)...)
			}
			candidates = append(candidates, scanNodeImages(value, file, nextPath, version, repo, repositories, values)...)
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			candidates = append(candidates, scanNodeImages(child, file, fmt.Sprintf("%s[%d]", path, i), version, repo, repositories, values)...)
		}
	case yaml.ScalarNode:
		if strings.Contains(node.Value, "image:") || strings.Contains(node.Value, "Image:") {
			rendered, err := renderImageTemplate(node.Value, values)
			if err == nil {
				for _, match := range embeddedImageLineRE.FindAllStringSubmatch(rendered, -1) {
					if len(match) < 3 || !isImageField(match[1]) {
						continue
					}
					for _, image := range expandImage(match[2], repo, repositories) {
						candidates = append(candidates, ImageCandidate{
							Image:      image,
							Repository: repo,
							File:       file,
							Field:      joinPath(path, match[1]),
							Version:    version,
						})
					}
				}
			}
		}
	}
	return candidates
}

func candidatesFromScalar(node *yaml.Node, file, field, version, repo string, repositories []string, values templateValues) []ImageCandidate {
	if node == nil || node.Kind != yaml.ScalarNode {
		return nil
	}
	rendered, err := renderImageTemplate(node.Value, values)
	if err != nil {
		return nil
	}
	var candidates []ImageCandidate
	for _, image := range expandImage(rendered, repo, repositories) {
		candidates = append(candidates, ImageCandidate{
			Image:      image,
			Repository: repo,
			File:       file,
			Field:      field,
			Version:    version,
		})
	}
	return candidates
}

func repositoryTagCandidate(node *yaml.Node, file, path, version, repo string, repositories []string, values templateValues) []ImageCandidate {
	fields := map[string]*yaml.Node{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		fields[node.Content[i].Value] = node.Content[i+1]
	}
	repoNode := fields["repository"]
	if repoNode == nil || repoNode.Kind != yaml.ScalarNode {
		return nil
	}
	tagNode := fields["tag"]
	if tagNode == nil {
		tagNode = fields["imageTag"]
	}
	if tagNode == nil || tagNode.Kind != yaml.ScalarNode {
		return nil
	}
	repoText, err := renderImageTemplate(repoNode.Value, values)
	if err != nil {
		return nil
	}
	tagText, err := renderImageTemplate(tagNode.Value, values)
	if err != nil {
		return nil
	}
	image := strings.TrimRight(cleanImage(repoText), "/") + ":" + cleanImage(tagText)
	if !looksLikeImage(image) {
		return nil
	}
	return []ImageCandidate{{
		Image:      image,
		Repository: repo,
		File:       file,
		Field:      joinPath(path, "repository:tag"),
		Version:    version,
	}}
}

func templateDefaults(data []byte) (map[string]any, map[string]any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	defaults := map[string]any{}
	globe := map[string]any{}
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err != nil {
			break
		}
		spec := mappingChild(&node, "spec")
		if spec == nil {
			continue
		}
		if necessary := mappingChild(spec, "necessary"); necessary != nil {
			mergeMaps(defaults, defaultsFromNode(necessary))
		}
		if globeNode := mappingChild(spec, "globe"); globeNode != nil {
			mergeMaps(globe, valuesFromNode(globeNode))
		}
	}
	return defaults, globe, nil
}

func mappingChild(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return mappingChild(node.Content[0], key)
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func mappingScalar(node *yaml.Node, key string) string {
	child := mappingChild(node, key)
	if child == nil || child.Kind != yaml.ScalarNode {
		return ""
	}
	return child.Value
}

func defaultsFromNode(node *yaml.Node) map[string]any {
	out := map[string]any{}
	if node.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		value := node.Content[i+1]
		switch value.Kind {
		case yaml.MappingNode:
			out[key] = defaultsFromNode(value)
		case yaml.ScalarNode:
			out[key] = defaultFromScalar(value.Value)
		}
	}
	return out
}

func defaultFromScalar(value string) any {
	if value == "" {
		return ""
	}
	var schema map[string]any
	if err := jsonUnmarshal(value, &schema); err == nil {
		if def, ok := schema["default"]; ok {
			return def
		}
		return ""
	}
	return value
}

func valuesFromNode(node *yaml.Node) map[string]any {
	var out map[string]any
	if err := node.Decode(&out); err != nil {
		return map[string]any{}
	}
	return out
}

func jsonUnmarshal(value string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func buildTemplateValues(meta PackageMetadata, defaults, globe map[string]any, version, repo string) templateValues {
	necessary := deepCopyMap(defaults)
	necessary["repository"] = repo
	if version != "" {
		necessary["version"] = version
	}

	renderGlobe := deepCopyMap(globe)
	registry, project := splitRepository(repo)
	renderGlobe["repository"] = registry
	renderGlobe["project"] = project
	renderGlobe["Name"] = "saola-image-export"
	renderGlobe["Namespace"] = "default"
	renderGlobe["PackageName"] = fmt.Sprintf("%s-%s", meta.Name, meta.Version)
	renderGlobe["Labels"] = map[string]any{}
	renderGlobe["Annotations"] = map[string]any{}

	return templateValues{
		Values:     map[string]any{},
		Globe:      renderGlobe,
		Necessary:  necessary,
		Parameters: map[string]any{},
		Step:       map[string]any{},
	}
}

func splitRepository(repo string) (string, string) {
	repo = strings.TrimRight(repo, "/")
	idx := strings.LastIndex(repo, "/")
	if idx < 0 {
		return repo, ""
	}
	return repo[:idx], repo[idx+1:]
}

func normalizeRepositories(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			repo := strings.TrimSpace(strings.TrimRight(part, "/"))
			if repo == "" || seen[repo] {
				continue
			}
			seen[repo] = true
			out = append(out, repo)
		}
	}
	return out
}

func expandImage(value, currentRepo string, repositories []string) []string {
	image := cleanImage(value)
	if !looksLikeImage(image) {
		return nil
	}
	if hasRepository(image) {
		return []string{image}
	}
	repos := repositories
	if currentRepo != "" {
		repos = []string{currentRepo}
	}
	var out []string
	for _, repo := range repos {
		out = append(out, strings.TrimRight(repo, "/")+"/"+image)
	}
	return out
}

func cleanImage(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = strings.TrimSpace(value)
	return value
}

func looksLikeImage(value string) bool {
	if value == "" || strings.Contains(value, "{{") || strings.Contains(value, "<no value>") {
		return false
	}
	if strings.ContainsAny(value, " \t\n\r") {
		return false
	}
	if strings.HasSuffix(value, "/") || strings.HasSuffix(value, ":") {
		return false
	}
	return strings.Contains(value, ":") || strings.Contains(value, "@sha256:")
}

func hasRepository(image string) bool {
	parts := strings.Split(image, "/")
	if len(parts) < 2 {
		return false
	}
	first := parts[0]
	return strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost"
}

func isImageField(key string) bool {
	lower := strings.ToLower(key)
	if strings.Contains(lower, "imagepull") {
		return false
	}
	return lower == "image" || strings.HasSuffix(lower, "image")
}

func logicalImageName(image string, repositories []string) string {
	for _, repo := range repositories {
		prefix := strings.TrimRight(repo, "/") + "/"
		if strings.HasPrefix(image, prefix) {
			return strings.TrimPrefix(image, prefix)
		}
	}
	return image
}

func appendUniqueCandidate(candidates []ImageCandidate, candidate ImageCandidate) []ImageCandidate {
	for _, existing := range candidates {
		if existing.Image == candidate.Image &&
			existing.File == candidate.File &&
			existing.Field == candidate.Field &&
			existing.Version == candidate.Version {
			return candidates
		}
	}
	return append(candidates, candidate)
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func repositoryOrder(repo string, repositories []string) int {
	for i, candidate := range repositories {
		if repo == candidate {
			return i
		}
	}
	return len(repositories)
}

func deepCopyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if nested, ok := v.(map[string]any); ok {
			out[k] = deepCopyMap(nested)
		} else {
			out[k] = v
		}
	}
	return out
}

func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}
