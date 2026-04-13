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
	"testing"
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
