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

package tarutil

import (
	"archive/tar"
	"bytes"
	"strings"
	"testing"
)

// makeTar creates a TAR archive in memory from entries.
func makeTar(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Size:     int64(len(e.content)),
			Typeflag: tar.TypeReg,
			Mode:     0o644,
		}
		if e.sizeOverride > 0 {
			hdr.Size = e.sizeOverride
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", e.name, err)
		}
		if _, err := tw.Write(e.content); err != nil {
			t.Fatalf("write content %q: %v", e.name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	return buf.Bytes()
}

type tarEntry struct {
	name         string
	content      []byte
	sizeOverride int64 // if >0, lie about size in header (for size-limit tests)
}

func TestReadTarInfo_ValidTar(t *testing.T) {
	data := makeTar(t, []tarEntry{
		{name: "pkg/metadata.yaml", content: []byte("name: test")},
		{name: "pkg/baselines/default.yaml", content: []byte("version: 1")},
	})

	info, err := ReadTarInfo(data)
	if err != nil {
		t.Fatalf("ReadTarInfo error: %v", err)
	}
	if len(info.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(info.Files))
	}
	if got := string(info.Files["metadata.yaml"]); got != "name: test" {
		t.Errorf("metadata.yaml = %q, want %q", got, "name: test")
	}
	if got := string(info.Files["baselines/default.yaml"]); got != "version: 1" {
		t.Errorf("baselines/default.yaml = %q, want %q", got, "version: 1")
	}
}

func TestReadTarInfo_SingleComponent(t *testing.T) {
	// Entry without "/" — should use the name as-is (len(dirs) < 2 branch).
	data := makeTar(t, []tarEntry{
		{name: "standalone.txt", content: []byte("hello")},
	})

	info, err := ReadTarInfo(data)
	if err != nil {
		t.Fatalf("ReadTarInfo error: %v", err)
	}
	if _, ok := info.Files["standalone.txt"]; !ok {
		t.Errorf("expected key 'standalone.txt', got keys: %v", fileKeys(info))
	}
}

func TestReadTarInfo_MaxFileSize(t *testing.T) {
	// Create a header that claims size > 100MB but provide minimal content.
	// ReadTarInfo should reject based on header size before reading body.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "pkg/huge.bin",
		Size:     maxFileSize + 1,
		Typeflag: tar.TypeReg,
		Mode:     0o644,
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	// Write a single byte so the tar is structurally valid enough.
	_, _ = tw.Write([]byte{0})
	_ = tw.Close()

	_, err := ReadTarInfo(buf.Bytes())
	if err == nil {
		t.Fatal("expected error for oversized file, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReadTarInfo_MaxFileCount(t *testing.T) {
	// Build a tar with maxFileCount+1 entries.
	entries := make([]tarEntry, maxFileCount+1)
	for i := range entries {
		entries[i] = tarEntry{
			name:    "pkg/" + strings.Repeat("a", 5) + string(rune('0'+i%10)) + ".txt",
			content: []byte("x"),
		}
	}
	// Use unique names to avoid overwriting in the map.
	for i := range entries {
		entries[i].name = "pkg/" + padInt(i) + ".txt"
	}
	data := makeTar(t, entries)

	_, err := ReadTarInfo(data)
	if err == nil {
		t.Fatal("expected error for too many files, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum file count") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTarInfo_ReadFile_Found(t *testing.T) {
	info := &TarInfo{
		Files: map[string][]byte{
			"baselines/default.yaml": []byte("content"),
		},
	}
	got, err := info.ReadFile("default.yaml")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != "content" {
		t.Errorf("ReadFile = %q, want %q", string(got), "content")
	}
}

func TestTarInfo_ReadFile_NotFound(t *testing.T) {
	info := &TarInfo{
		Files: map[string][]byte{
			"baselines/default.yaml": []byte("content"),
		},
	}
	_, err := info.ReadFile("nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// padInt returns a zero-padded string for creating unique tar entry names.
func padInt(n int) string {
	s := strings.Builder{}
	// Simple zero-padding to 6 digits.
	v := n
	digits := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		digits[i] = byte('0' + v%10)
		v /= 10
	}
	s.Write(digits)
	return s.String()
}

// fileKeys returns all keys from a TarInfo for diagnostic output.
func fileKeys(info *TarInfo) []string {
	keys := make([]string, 0, len(info.Files))
	for k := range info.Files {
		keys = append(keys, k)
	}
	return keys
}
