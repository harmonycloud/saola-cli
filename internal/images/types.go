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
	"io"
	"time"
)

// DefaultProbeTimeout is the per-image timeout used for probing image existence.
//
// DefaultProbeTimeout 是单个镜像存在性探测的默认超时时间。
const DefaultProbeTimeout = 30 * time.Second

// PackageMetadata contains the package fields needed by image export.
//
// PackageMetadata 保存镜像导出所需的包元数据字段。
type PackageMetadata struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
	App     App    `yaml:"app" json:"app"`
}

// App contains supported application versions.
//
// App 保存包支持的应用版本。
type App struct {
	Version []string `yaml:"version" json:"version"`
}

// ImageCandidate is one concrete image reference that can satisfy an image key.
//
// ImageCandidate 表示可以满足某个镜像项的一个具体候选镜像地址。
type ImageCandidate struct {
	Image      string `json:"image" yaml:"image"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	File       string `json:"file" yaml:"file"`
	Field      string `json:"field" yaml:"field"`
	Version    string `json:"version,omitempty" yaml:"version,omitempty"`
}

// ImageGroup groups repository candidates for the same logical image name.
//
// ImageGroup 将同一个逻辑镜像的多个仓库候选分组。
type ImageGroup struct {
	Name       string           `json:"name" yaml:"name"`
	Candidates []ImageCandidate `json:"candidates" yaml:"candidates"`
}

// ResolvedImage records the chosen image for an image group.
//
// ResolvedImage 记录一个镜像组最终命中的镜像。
type ResolvedImage struct {
	Name       string           `json:"name" yaml:"name"`
	Image      string           `json:"image" yaml:"image"`
	Repository string           `json:"repository,omitempty" yaml:"repository,omitempty"`
	File       string           `json:"file" yaml:"file"`
	Field      string           `json:"field" yaml:"field"`
	Version    string           `json:"version,omitempty" yaml:"version,omitempty"`
	Candidates []ImageCandidate `json:"candidates,omitempty" yaml:"candidates,omitempty"`
}

// ProbeError records why a concrete image candidate failed inspection.
//
// ProbeError 记录某个候选镜像探测失败的原因。
type ProbeError struct {
	Image      string `json:"image" yaml:"image"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	File       string `json:"file,omitempty" yaml:"file,omitempty"`
	Field      string `json:"field,omitempty" yaml:"field,omitempty"`
	Reason     string `json:"reason" yaml:"reason"`
	Message    string `json:"message" yaml:"message"`
}

// MissingImage records candidates that could not be resolved.
//
// MissingImage 记录未能命中的镜像候选。
type MissingImage struct {
	Name        string           `json:"name" yaml:"name"`
	Candidates  []ImageCandidate `json:"candidates" yaml:"candidates"`
	ProbeErrors []ProbeError     `json:"probeErrors,omitempty" yaml:"probeErrors,omitempty"`
}

// LockFile is written next to the exported image archive.
//
// LockFile 写入镜像导出归档旁边，用于记录实际命中的镜像。
type LockFile struct {
	Package      PackageMetadata `json:"package" yaml:"package"`
	GeneratedAt  time.Time       `json:"generatedAt" yaml:"generatedAt"`
	Repositories []string        `json:"repositories" yaml:"repositories"`
	Images       []ResolvedImage `json:"images" yaml:"images"`
	Missing      []MissingImage  `json:"missing,omitempty" yaml:"missing,omitempty"`
}

// ExportOptions controls image discovery, resolution, and export.
//
// ExportOptions 控制镜像发现、解析和导出行为。
type ExportOptions struct {
	PkgDir       string
	Output       string
	LockFile     string
	Repositories []string
	Platform     string
	MultiArch    bool
	Insecure     bool
	SkipMissing  bool
	DryRun       bool
	Timeout      time.Duration
	ProgressOut  io.Writer
	Runner       Runner
}
