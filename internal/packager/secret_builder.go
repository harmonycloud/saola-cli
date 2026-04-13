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
	"strings"

	zeusv1 "gitee.com/opensaola/opensaola/api/v1"
	saolaconsts "gitee.com/opensaola/saola-cli/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var immutable = true

// BuildInstallSecret constructs the corev1.Secret that opensaola watches to
// install a middleware package.
//
// name is the Secret name; if empty it defaults to "<meta.Name>-<meta.Version>".
// namespace is the target namespace (typically pkg-namespace / middleware-operator).
// meta is the parsed package metadata.
// data is the zstd-compressed TAR blob produced by PackDir.
//
// 构建 opensaola 用于监听安装的 corev1.Secret。
// name 为 Secret 名称，为空时默认使用 "<meta.Name>-<meta.Version>"。
// namespace 为目标命名空间（通常是 pkg-namespace / middleware-operator）。
// meta 为解析后的包元数据。
// data 为 PackDir 生成的 zstd 压缩 TAR 字节。
func BuildInstallSecret(name, namespace string, meta *Metadata, data []byte) *corev1.Secret {
	if name == "" {
		name = meta.Name + "-" + meta.Version
	}
	// Kubernetes requires lowercase RFC 1123 subdomain names.
	//
	// Kubernetes 要求 metadata.name 为小写 RFC 1123 子域名格式。
	name = strings.ToLower(name)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				zeusv1.LabelProject:        saolaconsts.ProjectOpenSaola,
				zeusv1.LabelComponent:      meta.Name,
				zeusv1.LabelPackageVersion: meta.Version,
				zeusv1.LabelPackageName:    name,
				zeusv1.LabelEnabled:        "false",
			},
			Annotations: map[string]string{
				zeusv1.LabelInstall: "true",
			},
		},
		Immutable: &immutable,
		Data: map[string][]byte{
			// Use saolaconsts.Release constant to keep the data key in sync with opensaola.
			//
			// 使用 saolaconsts.Release 常量，确保 data key 与 opensaola 保持同步。
			saolaconsts.Release: data,
		},
	}
}
