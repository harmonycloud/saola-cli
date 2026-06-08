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

package packages

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	saolaconsts "github.com/harmonycloud/saola-cli/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestCompressDecompress_Roundtrip verifies that compressing and then
// decompressing data produces the original input.
//
// TestCompressDecompress_Roundtrip 验证压缩后再解压能还原原始数据。
func TestCompressDecompress_Roundtrip(t *testing.T) {
	t.Parallel()
	original := []byte("test data for compression roundtrip")

	compressed, n, err := Compress(original)
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}
	if n != len(original) {
		t.Errorf("Compress wrote %d bytes, expected %d", n, len(original))
	}
	if len(compressed) == 0 {
		t.Fatal("Compress returned empty output")
	}

	decompressed, err := DeCompress(compressed)
	if err != nil {
		t.Fatalf("DeCompress returned error: %v", err)
	}
	if !bytes.Equal(decompressed, original) {
		t.Errorf("roundtrip mismatch: got %q, want %q", decompressed, original)
	}
}

// TestCompressDecompress_Empty verifies roundtrip with empty input.
//
// TestCompressDecompress_Empty 验证空输入的压缩解压往返。
func TestCompressDecompress_Empty(t *testing.T) {
	t.Parallel()
	original := []byte{}

	compressed, _, err := Compress(original)
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}

	decompressed, err := DeCompress(compressed)
	if err != nil {
		t.Fatalf("DeCompress returned error: %v", err)
	}
	if len(decompressed) != 0 {
		t.Errorf("expected empty output, got %d bytes", len(decompressed))
	}
}

// TestDeCompress_InvalidData verifies that DeCompress returns an error
// when given data that is not valid zstd compressed content.
//
// TestDeCompress_InvalidData 验证传入非法数据时 DeCompress 会返回错误。
func TestDeCompress_InvalidData(t *testing.T) {
	t.Parallel()
	_, err := DeCompress([]byte("not compressed data"))
	if err == nil {
		t.Fatal("expected error for invalid compressed data, got nil")
	}
}

// TestSetDataNamespace verifies that SetDataNamespace updates the global
// DataNamespace variable.
//
// TestSetDataNamespace 验证 SetDataNamespace 能正确更新全局 DataNamespace 变量。
func TestSetDataNamespace(t *testing.T) {
	// NOT parallel — modifies global DataNamespace.
	old := DataNamespace

	SetDataNamespace("custom-ns")
	if DataNamespace != "custom-ns" {
		t.Errorf("expected DataNamespace to be %q, got %q", "custom-ns", DataNamespace)
	}

	// Restore original value so other tests are not affected.
	SetDataNamespace(old)
	if DataNamespace != old {
		t.Errorf("expected DataNamespace restored to %q, got %q", old, DataNamespace)
	}
}

// TestList_UsesSecretMetadataWithoutDecompressing verifies that List stays
// lightweight and does not read/decompress package payloads.
//
// TestList_UsesSecretMetadataWithoutDecompressing 验证 List 只读取 Secret
// 元数据，不解压包内容。
func TestList_UsesSecretMetadataWithoutDecompressing(t *testing.T) {
	// NOT parallel — modifies global DataNamespace.
	old := DataNamespace
	SetDataNamespace("test-ns")
	defer SetDataNamespace(old)

	secret := packageMetadataObject("redis-v1", "test-ns", map[string]string{
		zeusv1.LabelProject:        saolaconsts.ProjectOpenSaola,
		zeusv1.LabelComponent:      "redis",
		zeusv1.LabelPackageVersion: "1.0.0",
		zeusv1.LabelEnabled:        "true",
	})
	cli := &metadataClient{objects: []*metav1.PartialObjectMetadata{secret}}

	pkgs, err := List(context.Background(), cli, Option{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	pkg := pkgs[0]
	if pkg.Name != "redis-v1" {
		t.Errorf("expected package name redis-v1, got %q", pkg.Name)
	}
	if pkg.Component != "redis" {
		t.Errorf("expected component redis, got %q", pkg.Component)
	}
	if pkg.Metadata == nil || pkg.Metadata.Name != "redis" || pkg.Metadata.Version != "1.0.0" {
		t.Fatalf("unexpected metadata: %#v", pkg.Metadata)
	}
	if !pkg.Enabled {
		t.Error("expected package to be enabled")
	}
	if len(pkg.Files) != 0 {
		t.Errorf("expected List to avoid loading files, got %d files", len(pkg.Files))
	}
}

// TestGetSummary_UsesSecretMetadataWithoutDecompressing verifies that GetSummary
// retrieves package metadata without reading/decompressing the package payload.
//
// TestGetSummary_UsesSecretMetadataWithoutDecompressing 验证 GetSummary 只读取
// Secret 元数据，不读取/解压包内容。
func TestGetSummary_UsesSecretMetadataWithoutDecompressing(t *testing.T) {
	// NOT parallel — modifies global DataNamespace.
	old := DataNamespace
	SetDataNamespace("test-ns")
	defer SetDataNamespace(old)

	secret := packageMetadataObject("redis-v1", "test-ns", map[string]string{
		zeusv1.LabelProject:        saolaconsts.ProjectOpenSaola,
		zeusv1.LabelComponent:      "redis",
		zeusv1.LabelPackageVersion: "1.0.0",
		zeusv1.LabelEnabled:        "true",
	})
	cli := &metadataClient{objects: []*metav1.PartialObjectMetadata{secret}}

	pkg, err := GetSummary(context.Background(), cli, "redis-v1")
	if err != nil {
		t.Fatalf("GetSummary returned error: %v", err)
	}
	if pkg.Name != "redis-v1" {
		t.Errorf("expected package name redis-v1, got %q", pkg.Name)
	}
	if pkg.Component != "redis" {
		t.Errorf("expected component redis, got %q", pkg.Component)
	}
	if pkg.Metadata == nil || pkg.Metadata.Name != "redis" || pkg.Metadata.Version != "1.0.0" {
		t.Fatalf("unexpected metadata: %#v", pkg.Metadata)
	}
	if !pkg.Enabled {
		t.Error("expected package to be enabled")
	}
	if len(pkg.Files) != 0 {
		t.Errorf("expected GetSummary to avoid loading files, got %d files", len(pkg.Files))
	}
}

func packageMetadataObject(name, namespace string, labels map[string]string) *metav1.PartialObjectMetadata {
	obj := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
	obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	return obj
}

type metadataClient struct {
	sigs.Client
	objects []*metav1.PartialObjectMetadata
}

func (c *metadataClient) Get(_ context.Context, key sigs.ObjectKey, obj sigs.Object, _ ...sigs.GetOption) error {
	target, ok := obj.(*metav1.PartialObjectMetadata)
	if !ok {
		return fmt.Errorf("unexpected get object type %T", obj)
	}
	for _, item := range c.objects {
		if item.Name == key.Name && item.Namespace == key.Namespace {
			*target = *item.DeepCopy()
			return nil
		}
	}
	return fmt.Errorf("metadata object %s/%s not found", key.Namespace, key.Name)
}

func (c *metadataClient) List(_ context.Context, list sigs.ObjectList, opts ...sigs.ListOption) error {
	target, ok := list.(*metav1.PartialObjectMetadataList)
	if !ok {
		return fmt.Errorf("unexpected list object type %T", list)
	}
	listOpts := &sigs.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(listOpts)
	}
	for _, item := range c.objects {
		if listOpts.Namespace != "" && item.Namespace != listOpts.Namespace {
			continue
		}
		if listOpts.LabelSelector != nil && !listOpts.LabelSelector.Matches(labels.Set(item.Labels)) {
			continue
		}
		target.Items = append(target.Items, *item.DeepCopy())
	}
	return nil
}

// TestCompressDecompress_LargeData verifies roundtrip with larger data.
//
// TestCompressDecompress_LargeData 验证较大数据的压缩解压往返。
func TestCompressDecompress_LargeData(t *testing.T) {
	t.Parallel()
	// Build a 64 KB test payload with repeating pattern.
	original := bytes.Repeat([]byte("abcdefghijklmnop"), 4096)

	compressed, _, err := Compress(original)
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}

	decompressed, err := DeCompress(compressed)
	if err != nil {
		t.Fatalf("DeCompress returned error: %v", err)
	}
	if !bytes.Equal(decompressed, original) {
		t.Errorf("roundtrip mismatch: lengths got %d, want %d", len(decompressed), len(original))
	}
}
