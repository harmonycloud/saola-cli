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

package consts

// LabelDefinition is the label key that records the baseline name on a CR.
// opensaola/internal/service/consts does not export this constant yet;
// once it does, replace all usages with the upstream definition.
//
// LabelDefinition 记录 CR 所使用的 baseline 名称的 label key。
// opensaola/internal/service/consts 中暂未导出该常量，等上游补充后统一替换。
const LabelDefinition = "middleware.cn/definition"

// ProjectOpenSaola is the project label value used by opensaola.
// Mirrors internal/service/consts.ProjectOpenSaola from the opensaola module
// which saola-cli cannot import (internal package).
//
// ProjectOpenSaola 是 opensaola 使用的 project label 值。
// 镜像自 opensaola 模块的 internal/service/consts.ProjectOpenSaola，
// saola-cli 无法直接导入 internal 包。
const ProjectOpenSaola = "opensaola"

// Release is the Secret data key for the zstd-compressed package TAR.
// Mirrors internal/service/packages.Release from the opensaola module.
//
// Release 是 Secret data 中 zstd 压缩包 TAR 的 key。
// 镜像自 opensaola 模块的 internal/service/packages.Release。
const Release = "package"
