package packager

import (
	"strings"

	"gitee.com/opensaola/opensaola/pkg/service/consts"
	"gitee.com/opensaola/opensaola/pkg/service/packages"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var immutable = true

// BuildInstallSecret constructs the corev1.Secret that zeus-operator watches to
// install a middleware package.
//
// name is the Secret name; if empty it defaults to "<meta.Name>-<meta.Version>".
// namespace is the target namespace (typically pkg-namespace / middleware-operator).
// meta is the parsed package metadata.
// data is the zstd-compressed TAR blob produced by PackDir.
//
// 构建 zeus-operator 用于监听安装的 corev1.Secret。
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
				consts.LabelProject:        consts.ProjectZeusOperator,
				consts.LabelComponent:      meta.Name,
				consts.LabelPackageVersion: meta.Version,
				consts.LabelPackageName:    name,
				consts.LabelEnabled:        "false",
			},
			Annotations: map[string]string{
				consts.LabelInstall: "true",
			},
		},
		Immutable: &immutable,
		Data: map[string][]byte{
			// Use packages.Release constant to keep the data key in sync with zeus-operator.
			//
			// 使用 packages.Release 常量，确保 data key 与 zeus-operator 保持同步。
			packages.Release: data,
		},
	}
}
