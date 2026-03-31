package packager

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitea.com/middleware-management/zeus-operator/pkg/service/packages"
	"gitea.com/middleware-management/zeus-operator/pkg/tools"
)

// makeDir creates a temporary directory populated with the given files (relative path → content).
//
// makeDir 在临时目录中创建指定文件，返回目录路径。
func makeDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdirall %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

// fileKeys returns a slice of keys from the files map, for readable error messages.
//
// fileKeys 返回 files 映射的所有 key，用于错误信息可读性。
func fileKeys(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	return keys
}

// tarEntryNames reads raw TAR bytes and returns all regular-file entry names.
//
// tarEntryNames 读取原始 TAR 字节，返回所有普通文件条目的名称。
func tarEntryNames(t *testing.T, rawTar []byte) []string {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(rawTar))
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read TAR entry: %v", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			names = append(names, hdr.Name)
		}
	}
	return names
}

// TestPackDir_NormalCase verifies that a well-formed package directory is packed, decompressed,
// and parsed correctly by zeus-operator's packages.DeCompress and tools.ReadTarInfo.
//
// TestPackDir_NormalCase 验证正常包目录可以被打包，并由 zeus-operator 工具正确解压和解析。
func TestPackDir_NormalCase(t *testing.T) {
	dir := makeDir(t, map[string]string{
		"metadata.yaml":              "name: myapp\nversion: \"1.0.0\"\nowner: team\ntype: middleware\n",
		"baselines/baseline.yaml":    "kind: MiddlewareBaseline\nname: baseline-1\n",
		"configurations/config.yaml": "kind: MiddlewareConfiguration\nname: config-1\n",
	})

	compressed, meta, err := PackDir(dir)
	if err != nil {
		t.Fatalf("PackDir returned unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("PackDir returned nil metadata")
	}
	if meta.Name != "myapp" || meta.Version != "1.0.0" {
		t.Errorf("metadata mismatch: got name=%q version=%q", meta.Name, meta.Version)
	}

	// Decompress with zeus-operator's DeCompress, then parse TAR structure.
	//
	// 用 zeus-operator 的 DeCompress 解压后，解析 TAR 结构。
	raw, err := packages.DeCompress(compressed)
	if err != nil {
		t.Fatalf("DeCompress failed: %v", err)
	}

	info, err := tools.ReadTarInfo(raw)
	if err != nil {
		t.Fatalf("ReadTarInfo failed: %v", err)
	}

	// After ReadTarInfo strips the root prefix, metadata.yaml and nested files must be present.
	//
	// ReadTarInfo 剥离根目录前缀后，metadata.yaml 及嵌套文件必须存在。
	if _, ok := info.Files["metadata.yaml"]; !ok {
		t.Errorf("expected metadata.yaml in TAR files, got: %v", fileKeys(info.Files))
	}
	if _, ok := info.Files["baselines/baseline.yaml"]; !ok {
		t.Errorf("expected baselines/baseline.yaml in TAR files, got: %v", fileKeys(info.Files))
	}
	if _, ok := info.Files["configurations/config.yaml"]; !ok {
		t.Errorf("expected configurations/config.yaml in TAR files, got: %v", fileKeys(info.Files))
	}
}

// TestPackDir_MissingMetadata verifies that PackDir returns an error when metadata.yaml is absent.
//
// TestPackDir_MissingMetadata 验证缺少 metadata.yaml 时 PackDir 返回错误。
func TestPackDir_MissingMetadata(t *testing.T) {
	dir := makeDir(t, map[string]string{
		"baselines/baseline.yaml": "kind: MiddlewareBaseline\n",
	})

	_, _, err := PackDir(dir)
	if err == nil {
		t.Fatal("expected error for missing metadata.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "metadata.yaml") {
		t.Errorf("expected error to mention metadata.yaml, got: %v", err)
	}
}

// TestPackDir_MissingName verifies that PackDir returns a validation error when name is absent.
//
// TestPackDir_MissingName 验证 metadata.yaml 缺少 name 字段时 PackDir 返回验证错误。
func TestPackDir_MissingName(t *testing.T) {
	dir := makeDir(t, map[string]string{
		"metadata.yaml": "version: \"1.0.0\"\n",
	})

	_, _, err := PackDir(dir)
	if err == nil {
		t.Fatal("expected validation error for missing name, got nil")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("expected error to mention 'name', got: %v", err)
	}
}

// TestPackDir_MissingVersion verifies that PackDir returns a validation error when version is absent.
//
// TestPackDir_MissingVersion 验证 metadata.yaml 缺少 version 字段时 PackDir 返回验证错误。
func TestPackDir_MissingVersion(t *testing.T) {
	dir := makeDir(t, map[string]string{
		"metadata.yaml": "name: myapp\n",
	})

	_, _, err := PackDir(dir)
	if err == nil {
		t.Fatal("expected validation error for missing version, got nil")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("expected error to mention 'version', got: %v", err)
	}
}

// TestPackDir_EmptyDir verifies that a directory with only metadata.yaml packs successfully.
//
// TestPackDir_EmptyDir 验证只含 metadata.yaml 的目录也能正常打包。
func TestPackDir_EmptyDir(t *testing.T) {
	dir := makeDir(t, map[string]string{
		"metadata.yaml": "name: bare\nversion: \"0.1.0\"\n",
	})

	compressed, meta, err := PackDir(dir)
	if err != nil {
		t.Fatalf("PackDir returned unexpected error: %v", err)
	}
	if meta.Name != "bare" {
		t.Errorf("unexpected metadata name: %q", meta.Name)
	}

	raw, err := packages.DeCompress(compressed)
	if err != nil {
		t.Fatalf("DeCompress failed: %v", err)
	}
	info, err := tools.ReadTarInfo(raw)
	if err != nil {
		t.Fatalf("ReadTarInfo failed: %v", err)
	}
	if _, ok := info.Files["metadata.yaml"]; !ok {
		t.Error("expected metadata.yaml in TAR files")
	}
}

// TestPackDir_TarRootPrefix verifies that every TAR entry is prefixed with "<name>-<version>/".
//
// TestPackDir_TarRootPrefix 验证 TAR 内所有条目均以 "<name>-<version>/" 为根目录前缀。
func TestPackDir_TarRootPrefix(t *testing.T) {
	const pkgName = "redis"
	const pkgVersion = "7.0.0"
	expectedPrefix := pkgName + "-" + pkgVersion + "/"

	dir := makeDir(t, map[string]string{
		"metadata.yaml":           "name: " + pkgName + "\nversion: \"" + pkgVersion + "\"\n",
		"configurations/cfg.yaml": "data: value\n",
	})

	compressed, _, err := PackDir(dir)
	if err != nil {
		t.Fatalf("PackDir failed: %v", err)
	}

	raw, err := packages.DeCompress(compressed)
	if err != nil {
		t.Fatalf("DeCompress failed: %v", err)
	}

	// Inspect raw TAR entries before ReadTarInfo strips the prefix.
	//
	// 在 ReadTarInfo 剥离前缀之前，直接检查原始 TAR 条目名称。
	names := tarEntryNames(t, raw)
	if len(names) == 0 {
		t.Fatal("TAR contains no regular-file entries")
	}
	for _, name := range names {
		if !strings.HasPrefix(name, expectedPrefix) {
			t.Errorf("TAR entry %q does not start with expected prefix %q", name, expectedPrefix)
		}
	}
}
