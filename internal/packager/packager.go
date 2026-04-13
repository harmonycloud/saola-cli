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

package packager

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gitee.com/opensaola/saola-cli/internal/packages"
	"gopkg.in/yaml.v3"
)

// App holds supported and deprecated application versions for a package.
//
// App 记录包所支持及已废弃的应用版本列表。
type App struct {
	Version           []string `yaml:"version" json:"version"`
	DeprecatedVersion []string `yaml:"deprecatedVersion" json:"deprecatedVersion"`
}

// Metadata mirrors packages.Metadata from opensaola for local use.
// Field layout must stay in sync with opensaola/internal/service/packages/packages.go.
//
// Metadata 镜像自 opensaola 的 packages.Metadata，用于本地解析。
// 字段布局必须与 opensaola/internal/service/packages/packages.go 保持一致。
type Metadata struct {
	Name        string `yaml:"name"  json:"name"`
	Version     string `yaml:"version" json:"version"`
	App         App    `yaml:"app" json:"app"`
	Owner       string `yaml:"owner" json:"owner"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
}

// Validate checks that mandatory fields are present.
//
// 检查必填字段是否存在。
func (m *Metadata) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("metadata.yaml: name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("metadata.yaml: version is required")
	}
	return nil
}

// PackDir packs the given directory into a zstd-compressed TAR byte slice.
//
// 将指定目录打包为 zstd 压缩的 TAR 字节切片。
//
// Expected directory layout:
//
//	<dir>/
//	  metadata.yaml
//	  baselines/
//	  configurations/
//	  actions/
//	  crds/
//
// The TAR root directory prefix is set to "<name>-<version>/" so that
// opensaola's ReadTarInfo can strip the first path component correctly.
//
// 预期目录结构见上。TAR 根目录前缀设置为 "<name>-<version>/"，以便 opensaola 的 ReadTarInfo 正确剥离首层路径。
func PackDir(dir string) (data []byte, meta *Metadata, err error) {
	// 1. Read and validate metadata.
	metaPath := filepath.Join(dir, "metadata.yaml")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read metadata.yaml: %w", err)
	}
	meta = &Metadata{}
	if err = yaml.Unmarshal(metaBytes, meta); err != nil {
		return nil, nil, fmt.Errorf("parse metadata.yaml: %w", err)
	}
	if err = meta.Validate(); err != nil {
		return nil, nil, err
	}

	// 2. Build the TAR in memory.
	prefix := fmt.Sprintf("%s-%s/", meta.Name, meta.Version)
	tarBuf := &bytes.Buffer{}
	tw := tar.NewWriter(tarBuf)

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Skip hidden entries (.git, .DS_Store, .gitignore, etc.).
		//
		// 跳过隐藏条目（.git、.DS_Store、.gitignore 等）。
		if d.Name() != "." && d.Name()[0] == '.' {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// Skip directories (TAR only stores regular files).
		//
		// 跳过目录（TAR 只存普通文件）。
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Normalise to forward slashes for cross-platform TAR compatibility.
		tarPath := prefix + filepath.ToSlash(rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name:    tarPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
			Typeflag: tar.TypeReg,
		}
		if err = tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write TAR header for %s: %w", rel, err)
		}

		fileBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}
		if _, err = tw.Write(fileBytes); err != nil {
			return fmt.Errorf("write TAR data for %s: %w", rel, err)
		}
		return nil
	})
	if walkErr != nil {
		return nil, nil, fmt.Errorf("walk directory: %w", walkErr)
	}
	if err = tw.Close(); err != nil {
		return nil, nil, fmt.Errorf("close TAR writer: %w", err)
	}

	// 3. Compress with zstd via opensaola's Compress().
	compressed, _, err := packages.Compress(tarBuf.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("compress: %w", err)
	}

	return compressed, meta, nil
}
