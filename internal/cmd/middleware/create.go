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

package middleware

import (
	"context"
	"fmt"
	"os"

	zeusv1 "github.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	saolaconsts "gitee.com/opensaola/saola-cli/internal/consts"
	zeusk8s "gitee.com/opensaola/saola-cli/internal/k8s"
	"gitee.com/opensaola/saola-cli/internal/packages"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	sigsyaml "sigs.k8s.io/yaml"
)

// CreateOptions holds parameters for "middleware create".
//
// CreateOptions 保存 middleware create 子命令的所有参数。
type CreateOptions struct {
	Config    *config.Config
	Namespace string
	File      string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；为 nil 时使用 client.New(cfg)。
	Client sigs.Client
}

// NewCmdCreate returns the middleware create command.
//
// 返回 middleware create 子命令。
func NewCmdCreate(cfg *config.Config) *cobra.Command {
	o := &CreateOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "create",
		Short: lang.T("从 YAML 文件创建 Middleware 资源", "Create a Middleware resource from a YAML file"),
		Long: lang.T(
			`从指定的 YAML 文件读取 Middleware 清单并在集群中创建。`,
			`Read a Middleware manifest from a YAML file and create it in the cluster.`,
		),
		Example: `  # 从清单文件创建 / Create from a manifest file
  saola middleware create -f middleware.yaml

  # 覆盖命名空间 / Override namespace
  saola middleware create -f middleware.yaml -n production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.File, "file", "f", "", lang.T("YAML 清单文件路径（必填）", "Path to a Middleware manifest YAML file (required)"))
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("覆盖清单中的 namespace", "Override namespace from the manifest"))
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

// Run executes the create logic.
//
// 执行 create 逻辑：读取 YAML、反序列化、可选覆盖 namespace，最后调用 k8s 创建。
func (o *CreateOptions) Run(ctx context.Context) error {
	// 1. Read the manifest file.
	//
	// 读取 YAML 文件内容。
	data, err := os.ReadFile(o.File)
	if err != nil {
		return fmt.Errorf("read file %s: %w", o.File, err)
	}

	// 2. Deserialise into Middleware using sigs.k8s.io/yaml so that json struct tags
	// (e.g. metadata.name) are respected, matching standard Kubernetes manifest format.
	//
	// 使用 sigs.k8s.io/yaml 反序列化，该库先转 JSON 再解析，
	// 能正确识别 json struct tag（如 metadata.name）。
	var mw zeusv1.Middleware
	if err = sigsyaml.Unmarshal(data, &mw); err != nil {
		return fmt.Errorf("parse middleware manifest: %w", err)
	}

	// 3. Override namespace when --namespace is explicitly provided.
	//
	// 若指定了 --namespace，则覆盖 manifest 中的 namespace。
	ns := o.Namespace
	if ns == "" {
		ns = o.Config.Namespace
	}
	if ns != "" {
		mw.Namespace = ns
	}
	// Middleware resources default to the "default" namespace when no namespace is specified,
	// matching kubectl behavior for namespaced resources.
	//
	// Middleware 资源在未指定 namespace 时默认使用 "default"，与 kubectl 对 namespaced 资源的行为一致。
	if mw.Namespace == "" {
		mw.Namespace = "default"
	}

	// 4. Create the resource via the k8s helper.
	//
	// 调用 k8s helper 创建资源。
	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	// 5. Auto-enrich labels and operatorBaseline required by opensaola.
	//
	// 自动补全 opensaola reconcile 所需的 labels 和 operatorBaseline 字段。
	if err = o.enrichMiddleware(ctx, cli, &mw); err != nil {
		return err
	}

	if err = zeusk8s.CreateMiddleware(ctx, cli, &mw); err != nil {
		return fmt.Errorf("create middleware %s/%s: %w", mw.Namespace, mw.Name, err)
	}

	fmt.Fprintf(os.Stdout, "middleware/%s created\n", mw.Name)

	// Warn if the Middleware needs a MiddlewareOperator but none exists yet.
	// Middleware does NOT auto-create MiddlewareOperator — they must be created independently.
	//
	// 若 Middleware 依赖 MiddlewareOperator 但目标命名空间中不存在，打印警告。
	// Middleware 不会自动创建 MiddlewareOperator，需独立创建。
	if mw.Spec.OperatorBaseline.Name != "" {
		if mw.Annotations == nil || mw.Annotations[zeusv1.LabelNoOperator] == "" {
			_, moErr := zeusk8s.GetMiddlewareOperator(ctx, cli, mw.Spec.OperatorBaseline.Name, mw.Namespace)
			if moErr != nil {
				fmt.Fprintf(os.Stderr,
					"warning: MiddlewareOperator %q not found in namespace %q; "+
						"create it with 'saola operator create' or the Middleware may not become Available\n",
					mw.Spec.OperatorBaseline.Name, mw.Namespace,
				)
			}
		}
	}

	return nil
}


// enrichMiddleware auto-completes the labels and operatorBaseline fields that
// opensaola requires to reconcile a Middleware CR successfully.
//
// It is a no-op when mw.Labels[zeusv1.LabelPackageName] is already set.
//
// enrichMiddleware 自动补全 opensaola reconcile 成功所需的 labels 与
// operatorBaseline 字段；若 LabelPackageName 已存在则跳过。
func (o *CreateOptions) enrichMiddleware(ctx context.Context, cli sigs.Client, mw *zeusv1.Middleware) error {
	// spec.baseline is mandatory; without it we cannot look up the matching package.
	//
	// spec.baseline 为必填项，缺少时无法定位对应包。
	if mw.Spec.Baseline == "" {
		return fmt.Errorf("spec.baseline is required")
	}

	// Skip auto-enrichment when the caller has already supplied the package label.
	//
	// 如果调用方已经设置了包名 label，则跳过自动补全。
	if mw.Labels != nil && mw.Labels[zeusv1.LabelPackageName] != "" {
		return nil
	}

	// Set the package namespace so that packages.Get* functions resolve Secrets
	// from the correct namespace.
	//
	// 设置包命名空间，确保 packages 系列函数从正确的命名空间读取 Secret。
	packages.SetDataNamespace(o.Config.PkgNamespace)

	// List all enabled package Secrets in the package namespace.
	//
	// 列出包命名空间中所有已启用的包 Secret。
	secrets, err := zeusk8s.GetSecrets(ctx, cli, o.Config.PkgNamespace, sigs.MatchingLabels{
		zeusv1.LabelProject: saolaconsts.ProjectOpenSaola,
		zeusv1.LabelEnabled: "true",
	})
	if err != nil {
		return fmt.Errorf("list package secrets: %w", err)
	}

	// Iterate over each package and look for a MiddlewareBaseline whose name
	// matches mw.Spec.Baseline.
	//
	// 遍历每个包，在其 MiddlewareBaseline 列表中查找名称匹配 mw.Spec.Baseline 的条目。
	var (
		matchedSecretName string
		matchedBaseline   *zeusv1.MiddlewareBaseline
	)
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		baselines, bErr := packages.GetMiddlewareBaselines(ctx, cli, secret.Name)
		if bErr != nil {
			// A parse error in one package should not block the whole operation;
			// log a warning and continue.
			//
			// 单个包解析失败不应阻断整个流程，打印警告后继续。
			fmt.Fprintf(os.Stderr, "warning: skip package %s: %v\n", secret.Name, bErr)
			continue
		}
		for _, bl := range baselines {
			if bl.Name == mw.Spec.Baseline {
				matchedSecretName = secret.Name
				matchedBaseline = bl
				break
			}
		}
		if matchedBaseline != nil {
			break
		}
	}

	if matchedBaseline == nil {
		return fmt.Errorf("no package found that contains MiddlewareBaseline %q; "+
			"ensure the package is installed and enabled in namespace %q",
			mw.Spec.Baseline, o.Config.PkgNamespace)
	}

	// Retrieve the package Secret again (it is already cached by packages.Get) to
	// read its labels for component and version.
	//
	// 重新获取包 Secret（已由 packages.Get 缓存）以读取 component 和 version label。
	pkgSecret, err := zeusk8s.GetSecret(ctx, cli, matchedSecretName, o.Config.PkgNamespace)
	if err != nil {
		return fmt.Errorf("get package secret %s: %w", matchedSecretName, err)
	}

	// Initialise Labels map if the manifest did not include any labels.
	//
	// 如果 manifest 未携带任何 label，初始化 Labels map。
	if mw.Labels == nil {
		mw.Labels = make(map[string]string)
	}

	// Set the four labels required by the opensaola reconciler.
	//
	// 设置 opensaola reconciler 所需的四个 label。
	mw.Labels[zeusv1.LabelPackageName] = matchedSecretName
	mw.Labels[zeusv1.LabelPackageVersion] = pkgSecret.Labels[zeusv1.LabelPackageVersion]
	mw.Labels[zeusv1.LabelComponent] = pkgSecret.Labels[zeusv1.LabelComponent]
	mw.Labels[saolaconsts.LabelDefinition] = mw.Spec.Baseline

	// Auto-fill spec.operatorBaseline only when the user has not set it explicitly.
	//
	// 仅在用户未显式设置 spec.operatorBaseline 时才自动填充。
	if mw.Spec.OperatorBaseline.Name == "" && matchedBaseline.Spec.OperatorBaseline.Name != "" {
		mw.Spec.OperatorBaseline = matchedBaseline.Spec.OperatorBaseline
	}

	// Print a summary of what was auto-completed so the user can verify.
	//
	// 打印自动补全摘要，便于用户核查。
	fmt.Fprintf(os.Stdout,
		"auto-enriched middleware %q:\n"+
			"  %s=%s\n"+
			"  %s=%s\n"+
			"  %s=%s\n"+
			"  %s=%s\n"+
			"  spec.operatorBaseline.name=%s spec.operatorBaseline.gvkName=%s\n",
		mw.Name,
		zeusv1.LabelPackageName, mw.Labels[zeusv1.LabelPackageName],
		zeusv1.LabelPackageVersion, mw.Labels[zeusv1.LabelPackageVersion],
		zeusv1.LabelComponent, mw.Labels[zeusv1.LabelComponent],
		saolaconsts.LabelDefinition, mw.Labels[saolaconsts.LabelDefinition],
		mw.Spec.OperatorBaseline.Name, mw.Spec.OperatorBaseline.GvkName,
	)

	return nil
}
