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

// Package tarutil provides TAR reading helpers.
// Mirrors opensaola/pkg/tools.ReadTarInfo which saola-cli cannot import
// due to broken import paths in that package.
//
// tarutil 包提供 TAR 读取辅助函数。
// 镜像自 opensaola/pkg/tools.ReadTarInfo，由于该包内部导入路径错误，
// saola-cli 无法直接导入。
package tarutil

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// maxFileSize is the maximum allowed size of a single file in the TAR archive (100MB).
	//
	// maxFileSize 是 TAR 归档中单个文件允许的最大大小（100MB）。
	maxFileSize = 100 * 1024 * 1024

	// maxFileCount is the maximum number of files allowed in a TAR archive.
	//
	// maxFileCount 是 TAR 归档中允许的最大文件数量。
	maxFileCount = 10000
)

// TarInfo holds the parsed contents of a TAR archive.
// Mirrors opensaola/pkg/tools.TarInfo.
//
// TarInfo 保存 TAR 归档的解析内容。
// 镜像自 opensaola/pkg/tools.TarInfo。
type TarInfo struct {
	Name  string            `json:"file_name"`
	Files map[string][]byte `json:"file_data"`
}

// ReadFile returns the content of the first file whose path contains the given name.
//
// ReadFile 返回路径包含给定 name 的第一个文件的内容。
func (t *TarInfo) ReadFile(name string) ([]byte, error) {
	for k, v := range t.Files {
		if strings.Contains(k, name) {
			return v, nil
		}
	}
	return nil, errors.New("file not found")
}

// ReadTarInfo reads a TAR archive from raw bytes, stripping the first path
// component from each entry (matching opensaola convention for package TARs).
//
// ReadTarInfo 从原始字节读取 TAR 归档，剥离每个条目的首层路径组件
// （与 opensaola 包 TAR 的惯例一致）。
func ReadTarInfo(data []byte) (*TarInfo, error) {
	info := &TarInfo{Files: make(map[string][]byte)}
	tr := tar.NewReader(bytes.NewBuffer(data))
	fileCount := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if hdr.Typeflag == tar.TypeReg {
			if hdr.Size > maxFileSize {
				return nil, fmt.Errorf("file %q exceeds maximum allowed size (%d > %d)", hdr.Name, hdr.Size, maxFileSize)
			}

			fileCount++
			if fileCount > maxFileCount {
				return nil, fmt.Errorf("archive exceeds maximum file count (%d)", maxFileCount)
			}

			dirs := strings.Split(hdr.Name, "/")
			var name string
			if len(dirs) < 2 {
				name = hdr.Name
			} else {
				name = strings.Join(dirs[1:], "/")
			}

			info.Files[name], err = io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
		}
	}
	return info, nil
}
