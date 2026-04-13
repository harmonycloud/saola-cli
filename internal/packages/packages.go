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

// Package packages provides package Secret parsing and baseline discovery
// for saola-cli. This is a local mirror of opensaola/internal/service/packages
// which saola-cli can no longer import after the pkg/ -> internal/ migration.
//
// packages 包提供包 Secret 的解析和 baseline 发现功能。
// 这是 opensaola/internal/service/packages 的本地镜像，
// 在 pkg/ -> internal/ 迁移后 saola-cli 无法再导入原包。
package packages

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	zeusv1 "github.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/tarutil"
	saolaconsts "gitee.com/opensaola/saola-cli/internal/consts"
	k8shelper "gitee.com/opensaola/saola-cli/internal/k8s"
	"github.com/klauspost/compress/zstd"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Release is the Secret data key for the zstd-compressed package TAR.
// Must stay in sync with opensaola/internal/service/packages.Release.
//
// Release 是 Secret data 中 zstd 压缩包 TAR 的 key，
// 必须与 opensaola/internal/service/packages.Release 保持一致。
const Release = saolaconsts.Release

var (
	// DataNamespace is the namespace where package Secrets live.
	//
	// DataNamespace 是包 Secret 所在的命名空间。
	DataNamespace = "default"
)

// SetDataNamespace sets the namespace used for all subsequent package lookups.
//
// SetDataNamespace 设置后续所有包查找使用的命名空间。
func SetDataNamespace(namespace string) {
	DataNamespace = namespace
}

// Metadata mirrors opensaola/internal/service/packages.Metadata.
// Field layout must stay in sync.
//
// Metadata 镜像自 opensaola/internal/service/packages.Metadata。
// 字段布局必须保持一致。
type Metadata struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	App         App    `json:"app"`
	Owner       string `json:"owner"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// App holds version information for the middleware application.
//
// App 保存中间件应用的版本信息。
type App struct {
	Version           []string `json:"version"`
	DeprecatedVersion []string `json:"deprecatedVersion"`
}

// Package mirrors opensaola/internal/service/packages.Package.
//
// Package 镜像自 opensaola/internal/service/packages.Package。
type Package struct {
	Name      string            `json:"name"`
	Created   string            `json:"created"`
	Files     map[string][]byte `json:"file"`
	Component string            `json:"component"`
	Metadata  *Metadata         `json:"metadata"`
	Enabled   bool              `json:"enabled"`
}

// Option mirrors opensaola/internal/service/packages.Option.
//
// Option 镜像自 opensaola/internal/service/packages.Option。
type Option struct {
	LabelComponent      string
	LabelPackageVersion string
}

// ---------------------------------------------------------------------------
// Core operations
// 核心操作
// ---------------------------------------------------------------------------

// List reads middleware packages matching the given option filters.
//
// List 读取符合筛选条件的中间件包列表。
func List(ctx context.Context, cli client.Client, opt Option) ([]*Package, error) {
	lbs := make(client.MatchingLabels)
	lbs[zeusv1.LabelProject] = saolaconsts.ProjectOpenSaola
	if opt.LabelComponent != "" {
		lbs[zeusv1.LabelComponent] = opt.LabelComponent
	}
	if opt.LabelPackageVersion != "" {
		lbs[zeusv1.LabelPackageVersion] = opt.LabelPackageVersion
	}
	secrets, err := k8shelper.GetSecrets(ctx, cli, DataNamespace, lbs)
	if err != nil {
		return nil, err
	}

	var pkgs []*Package
	for _, item := range secrets.Items {
		pkg, pErr := Get(ctx, cli, item.Name)
		if pErr != nil {
			return nil, pErr
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// Get retrieves and parses a single package Secret by name.
//
// Get 根据名称获取并解析单个包 Secret。
func Get(ctx context.Context, cli client.Client, name string) (*Package, error) {
	s, err := k8shelper.GetSecret(ctx, cli, name, DataNamespace)
	if err != nil {
		return nil, fmt.Errorf("get secret failed: %w", err)
	}
	return parseSecret(s)
}

// GetMetadata retrieves metadata from a package Secret.
//
// GetMetadata 从包 Secret 中获取元数据。
func GetMetadata(ctx context.Context, cli client.Client, packageName string) (*Metadata, error) {
	pkg, err := Get(ctx, cli, packageName)
	if err != nil {
		return nil, err
	}

	var metadata Metadata
	if err = yaml.Unmarshal(pkg.Files["metadata.yaml"], &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// ---------------------------------------------------------------------------
// Baseline discovery
// Baseline 发现
// ---------------------------------------------------------------------------

// GetMiddlewareBaselines returns all MiddlewareBaseline definitions from a package.
//
// GetMiddlewareBaselines 返回包中所有的 MiddlewareBaseline 定义。
func GetMiddlewareBaselines(ctx context.Context, cli client.Client, packageName string) ([]*zeusv1.MiddlewareBaseline, error) {
	pkg, err := Get(ctx, cli, packageName)
	if err != nil {
		return nil, err
	}

	var baselines []*zeusv1.MiddlewareBaseline
	for _, v := range pkg.Files {
		if bytes.Contains(v, []byte("kind: MiddlewareBaseline")) {
			b := new(zeusv1.MiddlewareBaseline)
			if uErr := yaml.Unmarshal(v, b); uErr != nil {
				return nil, uErr
			}
			baselines = append(baselines, b)
		}
	}
	return baselines, nil
}

// GetMiddlewareBaseline returns a single MiddlewareBaseline by name from a package.
//
// GetMiddlewareBaseline 从包中返回指定名称的单个 MiddlewareBaseline。
func GetMiddlewareBaseline(ctx context.Context, cli client.Client, name, packageName string) (*zeusv1.MiddlewareBaseline, error) {
	pkg, err := Get(ctx, cli, packageName)
	if err != nil {
		return nil, err
	}

	for _, v := range pkg.Files {
		if bytes.Contains(v, []byte("kind: MiddlewareBaseline")) {
			b := new(zeusv1.MiddlewareBaseline)
			if uErr := yaml.Unmarshal(v, b); uErr != nil {
				return nil, uErr
			}
			if b.Name == name {
				return b, nil
			}
		}
	}
	return nil, apiErrors.NewNotFound(schema.GroupResource{Group: "middleware.cn", Resource: "MiddlewareBaseline"}, name)
}

// GetMiddlewareOperatorBaselines returns all MiddlewareOperatorBaseline definitions from a package.
//
// GetMiddlewareOperatorBaselines 返回包中所有的 MiddlewareOperatorBaseline 定义。
func GetMiddlewareOperatorBaselines(ctx context.Context, cli client.Client, packageName string) ([]*zeusv1.MiddlewareOperatorBaseline, error) {
	pkg, err := Get(ctx, cli, packageName)
	if err != nil {
		return nil, err
	}

	var baselines []*zeusv1.MiddlewareOperatorBaseline
	for _, v := range pkg.Files {
		if bytes.Contains(v, []byte("kind: MiddlewareOperatorBaseline")) {
			b := new(zeusv1.MiddlewareOperatorBaseline)
			if uErr := yaml.Unmarshal(v, b); uErr != nil {
				return nil, uErr
			}
			baselines = append(baselines, b)
		}
	}
	return baselines, nil
}

// GetMiddlewareOperatorBaseline returns a single MiddlewareOperatorBaseline by name from a package.
//
// GetMiddlewareOperatorBaseline 从包中返回指定名称的单个 MiddlewareOperatorBaseline。
func GetMiddlewareOperatorBaseline(ctx context.Context, cli client.Client, name, packageName string) (*zeusv1.MiddlewareOperatorBaseline, error) {
	pkg, err := Get(ctx, cli, packageName)
	if err != nil {
		return nil, err
	}

	for _, v := range pkg.Files {
		if !bytes.Contains(v, []byte("kind: MiddlewareOperatorBaseline")) {
			continue
		}
		b := new(zeusv1.MiddlewareOperatorBaseline)
		if uErr := yaml.Unmarshal(v, b); uErr != nil {
			continue
		}
		if b.Name == name {
			return b, nil
		}
	}
	return nil, fmt.Errorf("MiddlewareOperatorBaseline %q not found in package %q", name, packageName)
}

// GetMiddlewareActionBaselines returns all MiddlewareActionBaseline definitions from a package.
//
// GetMiddlewareActionBaselines 返回包中所有的 MiddlewareActionBaseline 定义。
func GetMiddlewareActionBaselines(ctx context.Context, cli client.Client, packageName string) ([]*zeusv1.MiddlewareActionBaseline, error) {
	pkg, err := Get(ctx, cli, packageName)
	if err != nil {
		return nil, err
	}

	var baselines []*zeusv1.MiddlewareActionBaseline
	for _, v := range pkg.Files {
		if bytes.Contains(v, []byte("kind: MiddlewareActionBaseline")) {
			b := new(zeusv1.MiddlewareActionBaseline)
			if uErr := yaml.Unmarshal(v, b); uErr != nil {
				return nil, uErr
			}
			baselines = append(baselines, b)
		}
	}
	return baselines, nil
}

// ---------------------------------------------------------------------------
// Compression
// 压缩 / 解压
// ---------------------------------------------------------------------------

// Compress applies zstd compression to data.
//
// Compress 对数据进行 zstd 压缩。
func Compress(data []byte) ([]byte, int, error) {
	buf := bytes.NewBuffer([]byte{})
	w, err := zstd.NewWriter(buf, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, 0, err
	}
	num, err := w.Write(data)
	if err != nil {
		w.Close()
		return nil, 0, err
	}
	w.Close()
	return buf.Bytes(), num, nil
}

// DeCompress applies zstd decompression to data.
//
// DeCompress 对数据进行 zstd 解压。
func DeCompress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)
	decoder, err := zstd.NewReader(buf, zstd.IgnoreChecksum(true))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(decoder)
}

// ---------------------------------------------------------------------------
// Internal helpers
// 内部辅助函数
// ---------------------------------------------------------------------------

func parseSecret(s *corev1.Secret) (*Package, error) {
	decompressData, err := DeCompress(s.Data[Release])
	if err != nil {
		return nil, fmt.Errorf("decompress data failed: %w", err)
	}
	info, err := tarutil.ReadTarInfo(decompressData)
	if err != nil {
		return nil, fmt.Errorf("read tar info failed: %w", err)
	}

	var metadata Metadata
	if err = yaml.Unmarshal(info.Files["metadata.yaml"], &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata failed: %w", err)
	}

	return &Package{
		Name:      s.Name,
		Created:   s.CreationTimestamp.Format(time.DateTime),
		Files:     info.Files,
		Component: s.Labels[zeusv1.LabelComponent],
		Metadata:  &metadata,
		Enabled:   s.Labels[zeusv1.LabelEnabled] == "true",
	}, nil
}
