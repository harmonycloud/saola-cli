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

package operator

import (
	"context"
	"fmt"
	"os"

	zeusv1 "github.com/OpenSaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	saolaconsts "gitee.com/opensaola/saola-cli/internal/consts"
	zeusk8s "gitee.com/opensaola/saola-cli/internal/k8s"
	"gitee.com/opensaola/saola-cli/internal/packages"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	// sigsyaml recognises json struct tags, which is required for
	// correctly decoding Kubernetes ObjectMeta fields.
	//
	// sigsyaml 能识别 json struct tag，是正确解码 Kubernetes ObjectMeta 的必要条件。
	sigsyaml "sigs.k8s.io/yaml"
)

// CreateOptions holds parameters for "operator create".
//
// CreateOptions 保存 "operator create" 命令的参数。
type CreateOptions struct {
	Config    *config.Config
	Namespace string
	File      string
	// Client is injected in tests; nil means use client.New(cfg).
	//
	// Client 在测试中注入；nil 时使用 client.New(cfg) 创建。
	Client sigs.Client
}

// NewCmdCreate returns the "operator create" command.
//
// 返回 "operator create" 子命令。
func NewCmdCreate(cfg *config.Config) *cobra.Command {
	o := &CreateOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "create",
		Short: lang.T("从 YAML 文件创建 MiddlewareOperator 资源", "Create a MiddlewareOperator resource from a YAML file"),
		Long: lang.T(
			`从指定的 YAML 文件读取 MiddlewareOperator 清单并在集群中创建。`,
			`Read a MiddlewareOperator manifest from a YAML file and create it in the cluster.`,
		),
		Example: `  # 从清单文件创建 / Create from a manifest file
  saola operator create -f operator.yaml

  # 覆盖命名空间 / Override namespace
  saola operator create -f operator.yaml --namespace my-ns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", lang.T("覆盖清单中的命名空间", "Override the namespace from the manifest"))
	cmd.Flags().StringVarP(&o.File, "file", "f", "", lang.T("YAML 清单文件路径（必填）", "Path to a MiddlewareOperator manifest YAML file (required)"))
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

// Run executes the create logic.
//
// Run 执行创建逻辑。
func (o *CreateOptions) Run(ctx context.Context) error {
	// Read and unmarshal the manifest file.
	//
	// 读取并反序列化 YAML 文件。
	raw, err := os.ReadFile(o.File)
	if err != nil {
		return fmt.Errorf("read file %q: %w", o.File, err)
	}

	mo := &zeusv1.MiddlewareOperator{}
	if err = sigsyaml.Unmarshal(raw, mo); err != nil {
		return fmt.Errorf("unmarshal MiddlewareOperator: %w", err)
	}

	// --namespace overrides whatever is in the manifest.
	//
	// --namespace 覆盖 manifest 中的 namespace。
	if o.Namespace != "" {
		mo.Namespace = o.Namespace
	} else if o.Config.Namespace != "" && mo.Namespace == "" {
		mo.Namespace = o.Config.Namespace
	}

	if mo.Namespace == "" {
		return fmt.Errorf("namespace is required: specify --namespace or set it in the manifest")
	}

	// Use the injected client if provided, otherwise build one from config.
	//
	// 优先使用注入的 client，否则根据 config 创建。
	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	// Check for duplicates before creating.
	//
	// 创建前检查是否已存在同名资源。
	existing := &zeusv1.MiddlewareOperator{}
	getErr := cli.Get(ctx, sigs.ObjectKey{Name: mo.Name, Namespace: mo.Namespace}, existing)
	if getErr == nil {
		return fmt.Errorf("MiddlewareOperator %s/%s already exists", mo.Namespace, mo.Name)
	}
	if !apierrors.IsNotFound(getErr) {
		return fmt.Errorf("check existing MiddlewareOperator: %w", getErr)
	}

	// Auto-enrich labels required by opensaola before creating.
	//
	// 创建前自动补全 opensaola reconcile 所需的 labels。
	if err = o.enrichOperator(ctx, cli, mo); err != nil {
		return err
	}

	if err = cli.Create(ctx, mo); err != nil {
		return fmt.Errorf("create MiddlewareOperator: %w", err)
	}

	fmt.Fprintf(os.Stdout, "middlewareoperator/%s created\n", mo.Name)
	return nil
}


// enrichOperator auto-completes the four labels that opensaola requires to
// reconcile a MiddlewareOperator CR successfully.
//
// It is a no-op when mo.Spec.Baseline is empty (caller may have set labels manually)
// or when mo.Labels[zeusv1.LabelPackageName] is already set.
//
// enrichOperator 自动补全 opensaola reconcile 所需的四个 label；
// 若 spec.baseline 为空或 LabelPackageName 已存在则跳过。
func (o *CreateOptions) enrichOperator(ctx context.Context, cli sigs.Client, mo *zeusv1.MiddlewareOperator) error {
	// Defensively set the package namespace first, regardless of early-return
	// paths, so that any future packages.Get* call resolves from the correct ns.
	//
	// 防御性地先设置包命名空间，无论后续是否提前返回，确保 packages 系列函数
	// 始终从正确的命名空间读取 Secret。
	packages.SetDataNamespace(o.Config.PkgNamespace)

	// spec.baseline is empty — skip enrichment silently; the user may have
	// manually set all required labels.
	//
	// spec.baseline 为空时静默跳过；用户可能已手动设置了所有必要 label。
	if mo.Spec.Baseline == "" {
		return nil
	}

	// Skip auto-enrichment when the caller has already supplied the package label.
	//
	// 如果调用方已经设置了包名 label，则跳过自动补全。
	if mo.Labels != nil && mo.Labels[zeusv1.LabelPackageName] != "" {
		return nil
	}

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

	// Iterate over each package and look for a MiddlewareOperatorBaseline whose
	// name matches mo.Spec.Baseline.
	//
	// 遍历每个包，在其 MiddlewareOperatorBaseline 列表中查找名称匹配 mo.Spec.Baseline 的条目。
	var (
		matchedSecretName string
		matchedFound      bool
	)
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		baselines, bErr := packages.GetMiddlewareOperatorBaselines(ctx, cli, secret.Name)
		if bErr != nil {
			// A parse error in one package should not block the whole operation;
			// log a warning and continue.
			//
			// 单个包解析失败不应阻断整个流程，打印警告后继续。
			fmt.Fprintf(os.Stderr, "warning: skip package %s: %v\n", secret.Name, bErr)
			continue
		}
		for _, bl := range baselines {
			if bl.Name == mo.Spec.Baseline {
				matchedSecretName = secret.Name
				matchedFound = true
				break
			}
		}
		if matchedFound {
			break
		}
	}

	if !matchedFound {
		// No installed package contains the requested baseline — fail fast so the
		// user gets a clear error rather than a zombied MiddlewareOperator missing
		// the four labels that opensaola requires to reconcile successfully.
		//
		// 未找到包含目标 baseline 的已安装包时报错，避免 MiddlewareOperator 缺少
		// opensaola reconcile 所需的四个 label 而进入僵尸状态。
		return fmt.Errorf(
			"no installed package contains MiddlewareOperatorBaseline %q in namespace %q; "+
				"install the package first or set the four required labels manually",
			mo.Spec.Baseline, o.Config.PkgNamespace,
		)
	}

	// Retrieve the matched package Secret to read its component and version labels.
	//
	// 获取匹配的包 Secret，读取 component 和 version label。
	pkgSecret, err := zeusk8s.GetSecret(ctx, cli, matchedSecretName, o.Config.PkgNamespace)
	if err != nil {
		return fmt.Errorf("get package secret %s: %w", matchedSecretName, err)
	}

	// Initialise Labels map if the manifest did not include any labels.
	//
	// 如果 manifest 未携带任何 label，初始化 Labels map。
	if mo.Labels == nil {
		mo.Labels = make(map[string]string)
	}

	// Set the four labels required by the opensaola reconciler.
	//
	// 设置 opensaola reconciler 所需的四个 label。
	mo.Labels[zeusv1.LabelPackageName] = matchedSecretName
	mo.Labels[zeusv1.LabelPackageVersion] = pkgSecret.Labels[zeusv1.LabelPackageVersion]
	mo.Labels[zeusv1.LabelComponent] = pkgSecret.Labels[zeusv1.LabelComponent]
	mo.Labels[saolaconsts.LabelDefinition] = mo.Spec.Baseline

	// Print a summary of what was auto-completed so the user can verify.
	//
	// 打印自动补全摘要，便于用户核查。
	fmt.Fprintf(os.Stdout,
		"auto-enriched middlewareoperator %q:\n"+
			"  %s=%s\n"+
			"  %s=%s\n"+
			"  %s=%s\n"+
			"  %s=%s\n",
		mo.Name,
		zeusv1.LabelPackageName, mo.Labels[zeusv1.LabelPackageName],
		zeusv1.LabelPackageVersion, mo.Labels[zeusv1.LabelPackageVersion],
		zeusv1.LabelComponent, mo.Labels[zeusv1.LabelComponent],
		saolaconsts.LabelDefinition, mo.Labels[saolaconsts.LabelDefinition],
	)

	return nil
}
