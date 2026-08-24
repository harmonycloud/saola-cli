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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ociIndexMediaType       = "application/vnd.oci.image.index.v1+json"
	ociManifestMediaType    = "application/vnd.oci.image.manifest.v1+json"
	ociRefNameAnnotation    = "org.opencontainers.image.ref.name"
	ociImageConfigMediaType = "application/vnd.oci.image.config.v1+json"
)

type platformSpec struct {
	OS           string
	Architecture string
	Variant      string
}

type ociPlatform struct {
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
	Variant      string   `json:"variant,omitempty"`
}

type ociDescriptor struct {
	MediaType    string            `json:"mediaType,omitempty"`
	Digest       string            `json:"digest"`
	Size         int64             `json:"size"`
	URLs         []string          `json:"urls,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	Data         string            `json:"data,omitempty"`
	ArtifactType string            `json:"artifactType,omitempty"`
	Platform     *ociPlatform      `json:"platform,omitempty"`
}

type ociIndexDocument struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType,omitempty"`
	ArtifactType  string            `json:"artifactType,omitempty"`
	Manifests     []ociDescriptor   `json:"manifests"`
	Subject       *ociDescriptor    `json:"subject,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

type ociManifestDocument struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType,omitempty"`
	ArtifactType  string            `json:"artifactType,omitempty"`
	Config        ociDescriptor     `json:"config"`
	Layers        []ociDescriptor   `json:"layers"`
	Subject       *ociDescriptor    `json:"subject,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

type ociImageConfig struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"`
}

type stagedOCIPlatform struct {
	Ref      string
	Platform ociPlatform
}

func normalizeRequiredPlatforms(values []string) ([]platformSpec, error) {
	if len(values) == 0 {
		values = []string{"linux/amd64", "linux/arm64"}
	}

	seen := map[string]bool{}
	platforms := make([]platformSpec, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			platform, err := parsePlatformSpec(item)
			if err != nil {
				return nil, err
			}
			key := platformSpecString(platform)
			if seen[key] {
				continue
			}
			seen[key] = true
			platforms = append(platforms, platform)
		}
	}
	return platforms, nil
}

func parsePlatformSpec(value string) (platformSpec, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := strings.Split(value, "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" || (len(parts) == 3 && parts[2] == "") {
		return platformSpec{}, fmt.Errorf("invalid platform %q, expected os/arch[/variant]", value)
	}
	platform := platformSpec{OS: parts[0], Architecture: normalizeArchitecture(parts[1])}
	if len(parts) == 3 {
		platform.Variant = parts[2]
	}
	return platform, nil
}

func composeOCIMultiPlatformImage(layoutDir string, targetRef string, staged []stagedOCIPlatform) error {
	if len(staged) == 0 {
		return fmt.Errorf("no staged platforms for %s", targetRef)
	}
	indexPath := filepath.Join(layoutDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read OCI layout index: %w", err)
	}
	var index ociIndexDocument
	if err = json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("parse OCI layout index: %w", err)
	}

	stagedRefs := make(map[string]bool, len(staged))
	for _, item := range staged {
		stagedRefs[item.Ref] = true
	}
	childrenByRef := make(map[string]ociDescriptor, len(staged))
	remaining := make([]ociDescriptor, 0, len(index.Manifests)+1)
	for _, descriptor := range index.Manifests {
		ref := ""
		if descriptor.Annotations != nil {
			ref = descriptor.Annotations[ociRefNameAnnotation]
		}
		if stagedRefs[ref] {
			childrenByRef[ref] = descriptor
			continue
		}
		if ref != targetRef {
			remaining = append(remaining, descriptor)
		}
	}

	children := make([]ociDescriptor, 0, len(staged))
	for _, item := range staged {
		descriptor, ok := childrenByRef[item.Ref]
		if !ok {
			return fmt.Errorf("OCI layout is missing staged reference %s", item.Ref)
		}
		descriptor.Annotations = cloneAnnotationsWithoutRefName(descriptor.Annotations)
		platform := item.Platform
		descriptor.Platform = &platform
		children = append(children, descriptor)
	}

	imageIndex := ociIndexDocument{SchemaVersion: 2, MediaType: ociIndexMediaType, Manifests: children}
	encoded, err := json.Marshal(imageIndex)
	if err != nil {
		return err
	}
	digest, size, err := writeOCIBlob(layoutDir, encoded)
	if err != nil {
		return err
	}
	remaining = append(remaining, ociDescriptor{
		MediaType:   ociIndexMediaType,
		Digest:      digest,
		Size:        size,
		Annotations: map[string]string{ociRefNameAnnotation: targetRef},
	})
	index.Manifests = remaining
	encoded, err = json.Marshal(index)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(indexPath, encoded, 0o644)
}

func missingRequiredPlatforms(required []platformSpec, available []ociPlatform) []string {
	var missing []string
	for _, requested := range required {
		found := false
		for _, actual := range available {
			if platformMatches(requested, actual) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, platformSpecString(requested))
		}
	}
	return missing
}

func selectedPlatformNames(required []platformSpec, available []ociPlatform) []string {
	return ociPlatformNames(selectRequiredOCIPlatforms(required, available))
}

func requestedOCIPlatforms(required []platformSpec) []ociPlatform {
	platforms := make([]ociPlatform, 0, len(required))
	for _, requested := range required {
		platforms = append(platforms, ociPlatform{OS: requested.OS, Architecture: requested.Architecture, Variant: requested.Variant})
	}
	return platforms
}

func ociPlatformsFromNames(names []string) ([]ociPlatform, error) {
	platforms := make([]ociPlatform, 0, len(names))
	for _, name := range names {
		platform, err := parsePlatformSpec(name)
		if err != nil {
			return nil, err
		}
		platforms = append(platforms, ociPlatform{OS: platform.OS, Architecture: platform.Architecture, Variant: platform.Variant})
	}
	return uniqueOCIPlatforms(platforms), nil
}

func selectRequiredOCIPlatforms(required []platformSpec, available []ociPlatform) []ociPlatform {
	selected := make([]ociPlatform, 0, len(required))
	for _, requested := range required {
		for _, actual := range available {
			if platformMatches(requested, actual) {
				selected = append(selected, actual)
				break
			}
		}
	}
	return selected
}

func platformMatches(requested platformSpec, actual ociPlatform) bool {
	if requested.OS != strings.ToLower(actual.OS) || requested.Architecture != normalizeArchitecture(actual.Architecture) {
		return false
	}
	return requested.Variant == "" || requested.Variant == strings.ToLower(actual.Variant)
}

func platformSpecString(platform platformSpec) string {
	value := platform.OS + "/" + platform.Architecture
	if platform.Variant != "" {
		value += "/" + platform.Variant
	}
	return value
}

func ociPlatformString(platform ociPlatform) string {
	value := strings.ToLower(platform.OS) + "/" + normalizeArchitecture(platform.Architecture)
	if platform.Variant != "" {
		value += "/" + strings.ToLower(platform.Variant)
	}
	return value
}

func ociPlatformNames(platforms []ociPlatform) []string {
	unique := uniqueOCIPlatforms(platforms)
	names := make([]string, 0, len(unique))
	for _, platform := range unique {
		names = append(names, ociPlatformString(platform))
	}
	return names
}

func normalizeArchitecture(architecture string) string {
	switch strings.ToLower(architecture) {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		return strings.ToLower(architecture)
	}
}

func validOCIPlatform(platform *ociPlatform) bool {
	return platform != nil && platform.OS != "" && platform.Architecture != "" && platform.OS != "unknown" && platform.Architecture != "unknown"
}

func uniqueOCIPlatforms(platforms []ociPlatform) []ociPlatform {
	byName := make(map[string]ociPlatform, len(platforms))
	for _, platform := range platforms {
		if validOCIPlatform(&platform) {
			byName[ociPlatformString(platform)] = platform
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]ociPlatform, 0, len(names))
	for _, name := range names {
		result = append(result, byName[name])
	}
	return result
}

func formatOCIPlatforms(platforms []ociPlatform) string {
	names := ociPlatformNames(platforms)
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func cloneAnnotationsWithoutRefName(annotations map[string]string) map[string]string {
	if len(annotations) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(annotations))
	for key, value := range annotations {
		if key != ociRefNameAnnotation {
			cloned[key] = value
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func writeOCIBlob(layoutDir string, data []byte) (string, int64, error) {
	sum := sha256.Sum256(data)
	digest := fmt.Sprintf("sha256:%x", sum)
	path, err := ociBlobPath(layoutDir, digest)
	if err != nil {
		return "", 0, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", 0, err
	}
	if err = os.WriteFile(path, data, 0o644); err != nil {
		return "", 0, err
	}
	return digest, int64(len(data)), nil
}

func ociBlobPath(layoutDir string, digest string) (string, error) {
	algorithm, encoded, ok := strings.Cut(digest, ":")
	if !ok || algorithm == "" || encoded == "" || strings.ContainsAny(algorithm+encoded, `/\\`) {
		return "", fmt.Errorf("invalid OCI digest %q", digest)
	}
	return filepath.Join(layoutDir, "blobs", algorithm, encoded), nil
}
