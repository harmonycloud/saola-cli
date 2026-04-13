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

package create

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	zeusv1 "gitee.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/client"
	saolaconsts "gitee.com/opensaola/saola-cli/internal/consts"
	zeusk8s "gitee.com/opensaola/saola-cli/internal/k8s"
	"gitee.com/opensaola/saola-cli/internal/packages"
	"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/charmbracelet/huh"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
	sigsyaml "sigs.k8s.io/yaml"
)

// baselineEntry holds summary information about a discovered MiddlewareBaseline
// so we can present a selection list and later look up details.
//
// baselineEntry 保存已发现的 MiddlewareBaseline 的摘要信息，
// 用于构建选择列表，并在后续步骤中查找详细信息。
type baselineEntry struct {
	// Name is the MiddlewareBaseline resource name.
	// Name 是 MiddlewareBaseline 资源名称。
	Name string

	// DisplayLabel is the human-readable label shown in the selection form.
	// DisplayLabel 是在选择表单中展示的人类可读标签。
	DisplayLabel string

	// PackageSecretName is the name of the package Secret that contains this baseline.
	// PackageSecretName 是包含该 baseline 的包 Secret 名称。
	PackageSecretName string

	// PackageVersion is the version label value from the package Secret.
	// PackageVersion 是包 Secret 上的版本 label 值。
	PackageVersion string

	// Component is the component label value from the package Secret.
	// Component 是包 Secret 上的组件 label 值。
	Component string

	// Baseline is the full MiddlewareBaseline object.
	// Baseline 是完整的 MiddlewareBaseline 对象。
	Baseline *zeusv1.MiddlewareBaseline
}


// operatorBaselineEntry holds summary information about a discovered MiddlewareOperatorBaseline
// so we can present a selection list and later look up details.
//
// operatorBaselineEntry 保存已发现的 MiddlewareOperatorBaseline 的摘要信息，
// 用于构建选择列表，并在后续步骤中查找详细信息。
type operatorBaselineEntry struct {
	// Name is the MiddlewareOperatorBaseline resource name.
	// Name 是 MiddlewareOperatorBaseline 资源名称。
	Name string

	// DisplayLabel is the human-readable label shown in the selection form.
	// DisplayLabel 是在选择表单中展示的人类可读标签。
	DisplayLabel string

	// PackageSecretName is the name of the package Secret that contains this baseline.
	// PackageSecretName 是包含该 baseline 的包 Secret 名称。
	PackageSecretName string

	// PackageVersion is the version label value from the package Secret.
	// PackageVersion 是包 Secret 上的版本 label 值。
	PackageVersion string

	// Component is the component label value from the package Secret.
	// Component 是包 Secret 上的组件 label 值。
	Component string

	// Baseline is the full MiddlewareOperatorBaseline object, kept for accessing
	// spec fields such as Globe during the interactive flow.
	// Baseline 是完整的 MiddlewareOperatorBaseline 对象，用于在交互流程中访问
	// spec 字段（如 Globe）。
	Baseline *zeusv1.MiddlewareOperatorBaseline
}

// valueCollector pairs a dot-separated field path with a pointer to the string
// variable that the huh form will write the user's answer into.
//
// valueCollector 将点分隔字段路径与 huh 表单写入用户答案的字符串变量指针配对。
type valueCollector struct {
	path string
	ptr  *string
}

// RunInteractive runs the interactive resource creation flow.
// It first asks the user to choose a resource type (Middleware or MiddlewareOperator),
// then delegates to the appropriate sub-flow.
//
// RunInteractive 运行交互式资源创建流程。
// 首先询问用户要创建的资源类型（Middleware 或 MiddlewareOperator），
// 然后委托给对应的子流程处理。
func RunInteractive(ctx context.Context, cfg *config.Config, cli sigs.Client) error {
	// ----------------------------------------------------------------
	// Step 0: Select resource type.
	//
	// 步骤 0：选择资源类型。
	// ----------------------------------------------------------------
	var resourceKind string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(lang.T("选择资源类型", "Select Resource Type")).
				Options(
					huh.NewOption("Middleware", "Middleware"),
					huh.NewOption("MiddlewareOperator", "MiddlewareOperator"),
				).
				Value(&resourceKind),
		),
	).Run(); err != nil {
		return err
	}

	switch resourceKind {
	case "Middleware":
		return runInteractiveMiddleware(ctx, cfg, cli)
	case "MiddlewareOperator":
		return runInteractiveOperator(ctx, cfg, cli)
	default:
		return fmt.Errorf("%s", lang.T("未知的资源类型", "unknown resource kind"))
	}
}

// runInteractiveMiddleware runs the interactive Middleware creation flow.
// It guides the user through baseline selection, basic metadata, necessary field
// inputs, a YAML preview, and a final confirmation before creating the resource.
//
// runInteractiveMiddleware 运行交互式 Middleware 创建流程。
// 依次引导用户完成 baseline 选择、基本元数据填写、necessary 字段输入、
// YAML 预览，以及最终确认后创建资源。
func runInteractiveMiddleware(ctx context.Context, cfg *config.Config, cli sigs.Client) error {
	// ----------------------------------------------------------------
	// Step 1: Discover all enabled baselines across installed packages.
	//
	// 步骤 1：从所有已安装的启用包中发现可用的 baseline。
	// ----------------------------------------------------------------
	packages.SetDataNamespace(cfg.PkgNamespace)

	secrets, err := zeusk8s.GetSecrets(ctx, cli, cfg.PkgNamespace, sigs.MatchingLabels{
		zeusv1.LabelProject: saolaconsts.ProjectOpenSaola,
		zeusv1.LabelEnabled: "true",
	})
	if err != nil {
		return fmt.Errorf("list package secrets: %w", err)
	}

	var entries []baselineEntry
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		baselines, bErr := packages.GetMiddlewareBaselines(ctx, cli, secret.Name)
		if bErr != nil {
			fmt.Fprintf(os.Stderr, "warning: skip package %s: %v\n", secret.Name, bErr)
			continue
		}
		for _, bl := range baselines {
			label := bl.Name
			if bl.Annotations != nil && bl.Annotations["baselineName"] != "" {
				label = bl.Annotations["baselineName"] + " (" + bl.Name + ")"
			}
			entries = append(entries, baselineEntry{
				Name:              bl.Name,
				DisplayLabel:      label,
				PackageSecretName: secret.Name,
				PackageVersion:    secret.Labels[zeusv1.LabelPackageVersion],
				Component:         secret.Labels[zeusv1.LabelComponent],
				Baseline:          bl,
			})
		}
	}

	if len(entries) == 0 {
		return fmt.Errorf("no middleware baselines found in package namespace %q; "+
			"ensure at least one package is installed and enabled", cfg.PkgNamespace)
	}

	// ----------------------------------------------------------------
	// Step 2a: Select middleware component.
	// Group baselines by component label so the user first picks a
	// middleware type (e.g. Redis, PostgreSQL), then picks a baseline
	// within that component.
	//
	// 步骤 2a：选择中间件组件。
	// 按 component label 分组，用户先选中间件类型（如 Redis、PostgreSQL），
	// 再从该组件下选择 baseline。
	// ----------------------------------------------------------------
	componentSet := make(map[string]bool)
	for _, e := range entries {
		if e.Component != "" {
			componentSet[e.Component] = true
		}
	}

	var selectedComponent string
	if len(componentSet) > 1 {
		componentNames := make([]string, 0, len(componentSet))
		for c := range componentSet {
			componentNames = append(componentNames, c)
		}
		sort.Strings(componentNames)

		componentOptions := make([]huh.Option[string], 0, len(componentNames))
		for _, c := range componentNames {
			componentOptions = append(componentOptions, huh.NewOption(c, c))
		}

		if err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(lang.T("选择中间件", "Select Middleware")).
					Options(componentOptions...).
					Value(&selectedComponent),
			),
		).Run(); err != nil {
			return err
		}
	} else {
		// Only one component — auto-select it.
		//
		// 只有一个组件时自动选中。
		for c := range componentSet {
			selectedComponent = c
		}
	}

	// Filter baselines to the selected component.
	//
	// 按选中的组件过滤 baseline。
	var filtered []baselineEntry
	for _, e := range entries {
		if e.Component == selectedComponent {
			filtered = append(filtered, e)
		}
	}

	// ----------------------------------------------------------------
	// Step 2b: Select baseline within the chosen component.
	//
	// 步骤 2b：在选中的组件下选择 baseline。
	// ----------------------------------------------------------------
	var selectedBaselineName string
	baselineOptions := make([]huh.Option[string], 0, len(filtered))
	for _, e := range filtered {
		baselineOptions = append(baselineOptions, huh.NewOption(e.DisplayLabel, e.Name))
	}

	if len(filtered) == 1 {
		// Only one baseline for this component — auto-select it.
		//
		// 该组件下只有一个 baseline，自动选中。
		selectedBaselineName = filtered[0].Name
		fmt.Fprintf(os.Stdout, "auto-selected baseline: %s\n", filtered[0].DisplayLabel)
	} else {
		if err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(lang.T("选择 Baseline", "Select Baseline")).
					Options(baselineOptions...).
					Value(&selectedBaselineName),
			),
		).Run(); err != nil {
			return err
		}
	}

	// Find the full entry for the selected baseline.
	//
	// 根据选中名称找到完整的 entry。
	var selected *baselineEntry
	for i := range entries {
		if entries[i].Name == selectedBaselineName {
			selected = &entries[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("selected baseline %q not found", selectedBaselineName)
	}

	// ----------------------------------------------------------------
	// Step 3: Input basic metadata (name and namespace).
	//
	// 步骤 3：输入基本元数据（名称和命名空间）。
	// ----------------------------------------------------------------
	var name string
	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	if err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(lang.T("实例名称", "Instance Name")).
				Value(&name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New(lang.T("实例名称不能为空", "instance name is required"))
					}
					return nil
				}),
			huh.NewInput().
				Title(lang.T("命名空间", "Namespace")).
				Value(&namespace).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New(lang.T("命名空间不能为空", "namespace is required"))
					}
					return nil
				}),
		),
	).Run(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)

	// ----------------------------------------------------------------
	// Step 4: Build necessary fields form from baseline schema.
	//
	// 步骤 4：根据 baseline schema 构建 necessary 字段表单。
	// ----------------------------------------------------------------
	var necessaryRaw map[string]interface{}
	if len(selected.Baseline.Spec.Necessary.Raw) > 0 {
		if err = json.Unmarshal(selected.Baseline.Spec.Necessary.Raw, &necessaryRaw); err != nil {
			return fmt.Errorf("parse baseline necessary: %w", err)
		}
	}

	var collectors []valueCollector
	var fields []huh.Field

	if len(necessaryRaw) > 0 {
		schemas := ParseNecessarySchema(necessaryRaw)

		for _, s := range schemas {
			s := s // capture for closure

			// Initialise with the default value.
			//
			// 用默认值初始化。
			val := ""
			if s.Default != nil {
				val = fmt.Sprint(s.Default)
			}

			collectors = append(collectors, valueCollector{path: s.Path, ptr: &val})

			title := s.Label
			if title == "" {
				title = s.Path
			}

			switch s.Type {
			case "enum":
				opts := parseEnumOptions(s.Options)
				huhOpts := toHuhOptions(opts)
				// Ensure the default is a valid option; if not, leave val blank
				// so huh starts at the first item.
				//
				// 确保默认值在选项范围内；否则留空，huh 会从第一项开始。
				validDefault := false
				for _, o := range opts {
					if o == val {
						validDefault = true
						break
					}
				}
				if !validDefault && len(opts) > 0 {
					val = opts[0]
				}
				fields = append(fields, huh.NewSelect[string]().
					Title(title).
					Options(huhOpts...).
					Value(&val))

			case "version":
				// Retrieve app versions from package metadata.
				//
				// 从包 metadata 获取应用版本列表。
				meta, mErr := packages.GetMetadata(ctx, cli, selected.PackageSecretName)
				var versionOpts []huh.Option[string]
				if mErr == nil && meta != nil {
					for _, v := range getAppVersions(meta) {
						versionOpts = append(versionOpts, huh.NewOption(v, v))
					}
				}
				if len(versionOpts) == 0 {
					// Fallback to free-text input when no version list is available.
					//
					// 无版本列表时回退为自由文本输入。
					fields = append(fields, huh.NewInput().
						Title(title).
						Value(&val).
						Validate(requiredIfNeeded(s.Required)))
				} else {
					fields = append(fields, huh.NewSelect[string]().
						Title(title).
						Options(versionOpts...).
						Value(&val))
				}

			case "password":
				fields = append(fields, huh.NewInput().
					Title(title).
					EchoMode(huh.EchoModePassword).
					Value(&val).
					Validate(func(input string) error {
						if s.Required && strings.TrimSpace(input) == "" {
							return errors.New(lang.T("该字段为必填项", "this field is required"))
						}
						for _, pr := range s.Patterns {
							if pr.Pattern == "" {
								continue
							}
							matched, rErr := regexp.MatchString(pr.Pattern, input)
							if rErr != nil {
								return fmt.Errorf("invalid pattern %q: %w", pr.Pattern, rErr)
							}
							if !matched {
								return fmt.Errorf("%s", pr.Description)
							}
						}
						return nil
					}))

			case "storageClass":
				scNames, scErr := getStorageClasses(ctx, cli)
				if scErr != nil || len(scNames) == 0 {
					// Fallback to free-text when StorageClass list cannot be obtained.
					//
					// 无法获取 StorageClass 列表时回退为自由文本输入。
					fields = append(fields, huh.NewInput().
						Title(title).
						Value(&val).
						Validate(requiredIfNeeded(s.Required)))
				} else {
					scOpts := toHuhOptions(scNames)
					fields = append(fields, huh.NewSelect[string]().
						Title(title).
						Options(scOpts...).
						Value(&val))
				}

			case "int":
				fields = append(fields, huh.NewInput().
					Title(title).
					Value(&val).
					Validate(func(input string) error {
						if s.Required && strings.TrimSpace(input) == "" {
							return errors.New(lang.T("该字段为必填项", "this field is required"))
						}
						if input == "" {
							return nil
						}
						n, convErr := strconv.ParseFloat(input, 64)
						if convErr != nil {
							return errors.New(lang.T("请输入整数", "please enter an integer"))
						}
						if s.Min != 0 && n < s.Min {
							return fmt.Errorf("%s %v", lang.T("最小值为", "minimum value is"), s.Min)
						}
						if s.Max != 0 && n > s.Max {
							return fmt.Errorf("%s %v", lang.T("最大值为", "maximum value is"), s.Max)
						}
						return nil
					}))

			default: // "string" and any other unrecognised types
				fields = append(fields, huh.NewInput().
					Title(title).
					Value(&val).
					Validate(func(input string) error {
						if s.Required && strings.TrimSpace(input) == "" {
							return errors.New(lang.T("该字段为必填项", "this field is required"))
						}
						if s.Pattern != "" && input != "" {
							matched, rErr := regexp.MatchString(s.Pattern, input)
							if rErr != nil {
								return fmt.Errorf("invalid pattern %q: %w", s.Pattern, rErr)
							}
							if !matched {
								return fmt.Errorf(lang.T("输入不符合格式要求: %s", "input does not match required pattern: %s"), s.Pattern)
							}
						}
						return nil
					}))
			}
		}

		// Render all necessary fields in a single group.
		// Large schemas could be paginated in future, but a single group
		// is sufficient for the current baseline sizes.
		//
		// 将所有 necessary 字段渲染在一个分组里。
		// 大型 schema 将来可分页展示，当前规模单组足够。
		if len(fields) > 0 {
			if err = huh.NewForm(
				huh.NewGroup(fields...),
			).Run(); err != nil {
				return err
			}
		}
	}

	// Collect all captured values into a flat path→value map.
	//
	// 将所有捕获的值汇总到扁平 path→value map 中。
	flatValues := make(map[string]interface{}, len(collectors))
	for _, c := range collectors {
		flatValues[c.path] = *c.ptr
	}

	// ----------------------------------------------------------------
	// Step 5: Build the Middleware object and show a YAML preview.
	//
	// 步骤 5：构建 Middleware 对象并展示 YAML 预览。
	// ----------------------------------------------------------------
	necessaryNested := BuildNecessaryValues(flatValues)

	necessaryBytes, err := json.Marshal(necessaryNested)
	if err != nil {
		return fmt.Errorf("marshal necessary: %w", err)
	}

	mw := &zeusv1.Middleware{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "middleware.cn/v1",
			Kind:       "Middleware",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				zeusv1.LabelPackageName:    selected.PackageSecretName,
				zeusv1.LabelPackageVersion: selected.PackageVersion,
				zeusv1.LabelComponent:      selected.Component,
				saolaconsts.LabelDefinition:            selected.Name,
			},
		},
		Spec: zeusv1.MiddlewareSpec{
			Baseline:         selected.Name,
			OperatorBaseline: selected.Baseline.Spec.OperatorBaseline,
			Necessary:        runtime.RawExtension{Raw: necessaryBytes},
		},
	}

	yamlBytes, err := sigsyaml.Marshal(mw)
	if err != nil {
		return fmt.Errorf("marshal preview yaml: %w", err)
	}
	fmt.Fprintf(os.Stdout, "\n--- %s / Preview ---\n%s\n", lang.T("预览", "Preview"), string(yamlBytes))

	// ----------------------------------------------------------------
	// Step 6: Confirm and create.
	//
	// 步骤 6：确认并创建资源。
	// ----------------------------------------------------------------
	var confirm bool
	if err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(lang.T("确认创建？", "Confirm creation?")).
				Value(&confirm),
		),
	).Run(); err != nil {
		return err
	}

	if !confirm {
		fmt.Fprintln(os.Stdout, lang.T("已取消", "Cancelled"))
		return nil
	}

	if err = zeusk8s.CreateMiddleware(ctx, cli, mw); err != nil {
		return fmt.Errorf("create middleware %s/%s: %w", namespace, name, err)
	}

	fmt.Fprintf(os.Stdout, "middleware/%s created\n", name)
	return nil
}

// runInteractiveOperator runs the interactive MiddlewareOperator creation flow.
// It discovers all MiddlewareOperatorBaselines from installed packages, guides the user
// through baseline selection and basic metadata input, then creates the resource.
//
// runInteractiveOperator 运行交互式 MiddlewareOperator 创建流程。
// 从已安装包中发现所有 MiddlewareOperatorBaseline，引导用户选择 baseline
// 并填写基本元数据，最终创建资源。
func runInteractiveOperator(ctx context.Context, cfg *config.Config, cli sigs.Client) error {
	// ----------------------------------------------------------------
	// Step 1: Discover all MiddlewareOperatorBaselines from enabled packages.
	//
	// 步骤 1：从所有已安装启用包中发现 MiddlewareOperatorBaseline。
	// ----------------------------------------------------------------
	packages.SetDataNamespace(cfg.PkgNamespace)

	secrets, err := zeusk8s.GetSecrets(ctx, cli, cfg.PkgNamespace, sigs.MatchingLabels{
		zeusv1.LabelProject: saolaconsts.ProjectOpenSaola,
		zeusv1.LabelEnabled: "true",
	})
	if err != nil {
		return fmt.Errorf("list package secrets: %w", err)
	}

	var opEntries []operatorBaselineEntry
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		baselines, bErr := packages.GetMiddlewareOperatorBaselines(ctx, cli, secret.Name)
		if bErr != nil {
			fmt.Fprintf(os.Stderr, "warning: skip package %s: %v\n", secret.Name, bErr)
			continue
		}
		for _, bl := range baselines {
			// Use annotation "baselineName" as display label when available,
			// consistent with the Middleware baseline discovery logic.
			//
			// 有 "baselineName" 注解时用作展示标签，与 Middleware baseline 发现逻辑保持一致。
			label := bl.Name
			if bl.Annotations != nil && bl.Annotations["baselineName"] != "" {
				label = bl.Annotations["baselineName"] + " (" + bl.Name + ")"
			}
			opEntries = append(opEntries, operatorBaselineEntry{
				Name:              bl.Name,
				DisplayLabel:      label,
				PackageSecretName: secret.Name,
				PackageVersion:    secret.Labels[zeusv1.LabelPackageVersion],
				Component:         secret.Labels[zeusv1.LabelComponent],
				Baseline:          bl,
			})
		}
	}

	if len(opEntries) == 0 {
		return fmt.Errorf("no middleware operator baselines found in package namespace %q; "+
			"ensure at least one package is installed and enabled", cfg.PkgNamespace)
	}

	// ----------------------------------------------------------------
	// Step 2a: Select middleware component.
	//
	// 步骤 2a：选择中间件组件。
	// ----------------------------------------------------------------
	opComponentSet := make(map[string]bool)
	for _, e := range opEntries {
		if e.Component != "" {
			opComponentSet[e.Component] = true
		}
	}

	var selectedComponent string
	if len(opComponentSet) > 1 {
		componentNames := make([]string, 0, len(opComponentSet))
		for c := range opComponentSet {
			componentNames = append(componentNames, c)
		}
		sort.Strings(componentNames)

		componentOptions := make([]huh.Option[string], 0, len(componentNames))
		for _, c := range componentNames {
			componentOptions = append(componentOptions, huh.NewOption(c, c))
		}

		if err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(lang.T("选择中间件", "Select Middleware")).
					Options(componentOptions...).
					Value(&selectedComponent),
			),
		).Run(); err != nil {
			return err
		}
	} else {
		for c := range opComponentSet {
			selectedComponent = c
		}
	}

	var filteredOp []operatorBaselineEntry
	for _, e := range opEntries {
		if e.Component == selectedComponent {
			filteredOp = append(filteredOp, e)
		}
	}

	// ----------------------------------------------------------------
	// Step 2b: Select OperatorBaseline within the chosen component.
	//
	// 步骤 2b：在选中的组件下选择 OperatorBaseline。
	// ----------------------------------------------------------------
	var selectedBaselineName string
	baselineOptions := make([]huh.Option[string], 0, len(filteredOp))
	for _, e := range filteredOp {
		baselineOptions = append(baselineOptions, huh.NewOption(e.DisplayLabel, e.Name))
	}

	if len(filteredOp) == 1 {
		selectedBaselineName = filteredOp[0].Name
		fmt.Fprintf(os.Stdout, "auto-selected operator baseline: %s\n", filteredOp[0].DisplayLabel)
	} else {
		if err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(lang.T("选择 OperatorBaseline", "Select OperatorBaseline")).
					Options(baselineOptions...).
					Value(&selectedBaselineName),
			),
		).Run(); err != nil {
			return err
		}
	}

	// Find the full entry for the selected baseline.
	//
	// 根据选中名称找到完整的 entry。
	var selected *operatorBaselineEntry
	for i := range opEntries {
		if opEntries[i].Name == selectedBaselineName {
			selected = &opEntries[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("selected operator baseline %q not found", selectedBaselineName)
	}

	// ----------------------------------------------------------------
	// Step 3: Input basic metadata (name and namespace).
	//
	// 步骤 3：输入基本元数据（名称和命名空间）。
	// ----------------------------------------------------------------
	var name string
	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	if err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(lang.T("实例名称", "Instance Name")).
				Value(&name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New(lang.T("实例名称不能为空", "instance name is required"))
					}
					return nil
				}),
			huh.NewInput().
				Title(lang.T("命名空间", "Namespace")).
				Value(&namespace).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New(lang.T("命名空间不能为空", "namespace is required"))
					}
					return nil
				}),
		),
	).Run(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	namespace = strings.TrimSpace(namespace)

	// ----------------------------------------------------------------
	// Step 3.5: Optionally configure Globe (image registry overrides).
	// Read the baseline's spec.globe, present each key as an editable
	// input pre-filled with the baseline default. If the user changes
	// any value the whole map is written to mo.spec.globe; if every
	// value matches the baseline default, globe is left nil so the
	// controller inherits it automatically.
	//
	// 步骤 3.5：可选配置 Globe（镜像仓库覆盖）。
	// 读取 baseline 的 spec.globe，将每个 key 展示为可编辑输入框，
	// 预填 baseline 默认值。若用户修改了任意值则将整个 map 写入
	// mo.spec.globe；若所有值与默认值一致则留 nil，由 controller 自动继承。
	// ----------------------------------------------------------------
	var moGlobe *runtime.RawExtension

	if selected.Baseline != nil &&
		selected.Baseline.Spec.Globe != nil &&
		len(selected.Baseline.Spec.Globe.Raw) > 0 {

		// Deserialise the baseline globe into a flat string map.
		//
		// 将 baseline globe 反序列化为扁平字符串 map。
		var baselineGlobeRaw map[string]interface{}
		if err = json.Unmarshal(selected.Baseline.Spec.Globe.Raw, &baselineGlobeRaw); err != nil {
			return fmt.Errorf("parse baseline globe: %w", err)
		}

		// Sort keys for deterministic rendering order.
		//
		// 对 key 排序以保证渲染顺序确定。
		globeKeys := make([]string, 0, len(baselineGlobeRaw))
		for k := range baselineGlobeRaw {
			globeKeys = append(globeKeys, k)
		}
		sort.Strings(globeKeys)

		// Build one huh.Input per key, defaulting to the baseline value.
		// Map elements cannot be addressed in Go, so we store values in a
		// parallel slice of strings and sync back after the form runs.
		//
		// 为每个 key 构建一个 huh.Input，默认值来自 baseline。
		// Go 中 map 元素不可取地址，因此用平行字符串切片存值，
		// 表单运行后再同步回来。
		globeDefaults := make([]string, len(globeKeys))
		globeUserVals := make([]string, len(globeKeys))
		globeFields := make([]huh.Field, 0, len(globeKeys))
		for i, k := range globeKeys {
			defaultVal := fmt.Sprint(baselineGlobeRaw[k])
			globeDefaults[i] = defaultVal
			globeUserVals[i] = defaultVal
			globeFields = append(globeFields, huh.NewInput().
				Title(k).
				Value(&globeUserVals[i]))
		}

		if err = huh.NewForm(
			huh.NewGroup(globeFields...).
				Title(lang.T("镜像仓库配置 / Image Repository Configuration", "Image Repository Configuration / 镜像仓库配置")),
		).Run(); err != nil {
			return err
		}

		// Compare each final value against the baseline default.
		// Only set mo.spec.globe when at least one value differs.
		//
		// 逐 key 比对最终值与 baseline 默认值。
		// 仅当至少一个值发生变化时才设置 mo.spec.globe。
		changed := false
		for i := range globeKeys {
			if globeUserVals[i] != globeDefaults[i] {
				changed = true
				break
			}
		}

		if changed {
			// Rebuild the globe map using the user-provided values and serialise.
			//
			// 用用户输入值重建 globe map 并序列化。
			finalGlobe := make(map[string]interface{}, len(globeKeys))
			for i, k := range globeKeys {
				finalGlobe[k] = globeUserVals[i]
			}
			globeBytes, mErr := json.Marshal(finalGlobe)
			if mErr != nil {
				return fmt.Errorf("marshal globe: %w", mErr)
			}
			moGlobe = &runtime.RawExtension{Raw: globeBytes}
		}
	}

	// ----------------------------------------------------------------
	// Step 4: Build MiddlewareOperator object with labels auto-populated.
	// Other spec fields (PreActions, Permissions, Deployment,
	// Configurations) are intentionally omitted here; the controller
	// will inherit them from the referenced OperatorBaseline.
	// Globe is set only when the user explicitly overrode a value above.
	//
	// 步骤 4：构建 MiddlewareOperator 对象并自动补全 labels。
	// 其余 spec 字段（PreActions、Permissions、Deployment、
	// Configurations）有意留空，controller 会从 OperatorBaseline 继承。
	// Globe 仅在用户在上一步显式修改时才设置。
	// ----------------------------------------------------------------
	mo := &zeusv1.MiddlewareOperator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "middleware.cn/v1",
			Kind:       "MiddlewareOperator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				zeusv1.LabelPackageName:    selected.PackageSecretName,
				zeusv1.LabelPackageVersion: selected.PackageVersion,
				zeusv1.LabelComponent:      selected.Component,
				saolaconsts.LabelDefinition:            selected.Name,
			},
		},
		Spec: zeusv1.MiddlewareOperatorSpec{
			Baseline: selected.Name,
			Globe:    moGlobe,
		},
	}

	// ----------------------------------------------------------------
	// Step 5: Show YAML preview.
	//
	// 步骤 5：展示 YAML 预览。
	// ----------------------------------------------------------------
	yamlBytes, err := sigsyaml.Marshal(mo)
	if err != nil {
		return fmt.Errorf("marshal preview yaml: %w", err)
	}
	fmt.Fprintf(os.Stdout, "\n--- %s / Preview ---\n%s\n", lang.T("预览", "Preview"), string(yamlBytes))

	// ----------------------------------------------------------------
	// Step 6: Confirm and create.
	//
	// 步骤 6：确认并创建资源。
	// ----------------------------------------------------------------
	var confirm bool
	if err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(lang.T("确认创建？", "Confirm creation?")).
				Value(&confirm),
		),
	).Run(); err != nil {
		return err
	}

	if !confirm {
		fmt.Fprintln(os.Stdout, lang.T("已取消", "Cancelled"))
		return nil
	}

	// Check for duplicate before creating, consistent with operator/create.go.
	//
	// 创建前检查是否已存在同名资源，与 operator/create.go 保持一致。
	existing := &zeusv1.MiddlewareOperator{}
	getErr := cli.Get(ctx, sigs.ObjectKey{Name: name, Namespace: namespace}, existing)
	if getErr == nil {
		return fmt.Errorf("MiddlewareOperator %s/%s already exists", namespace, name)
	}
	if !apierrors.IsNotFound(getErr) {
		return fmt.Errorf("check existing MiddlewareOperator: %w", getErr)
	}

	if err = cli.Create(ctx, mo); err != nil {
		return fmt.Errorf("create MiddlewareOperator %s/%s: %w", namespace, name, err)
	}

	fmt.Fprintf(os.Stdout, "middlewareoperator/%s created\n", name)
	return nil
}

// newInteractiveCli resolves the k8s client to use in RunInteractive.
// If injected is non-nil it is returned as-is; otherwise a live client is built
// from cfg.
//
// newInteractiveCli 解析 RunInteractive 中使用的 k8s 客户端。
// 若 injected 非 nil 则直接返回；否则根据 cfg 创建真实客户端。
func newInteractiveCli(cfg *config.Config, injected sigs.Client) (sigs.Client, error) {
	if injected != nil {
		return injected, nil
	}
	return client.New(cfg).Get()
}

// ----------------------------------------------------------------
// Helper functions
// ----------------------------------------------------------------

// parseEnumOptions splits a comma-separated options string into a trimmed slice.
//
// parseEnumOptions 将逗号分隔的 options 字符串拆分为去空格后的字符串切片。
func parseEnumOptions(s string) []string {
	raw := strings.Split(s, ",")
	result := make([]string, 0, len(raw))
	for _, o := range raw {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// toHuhOptions converts a string slice into a slice of huh.Option[string].
//
// toHuhOptions 将字符串切片转换为 huh.Option[string] 切片。
func toHuhOptions(opts []string) []huh.Option[string] {
	result := make([]huh.Option[string], 0, len(opts))
	for _, o := range opts {
		result = append(result, huh.NewOption(o, o))
	}
	return result
}

// getStorageClasses lists StorageClass names from the cluster, sorted alphabetically.
// Returns an error when the cluster cannot be reached or the resource type is
// not registered in the client's scheme.
//
// getStorageClasses 从集群获取 StorageClass 名称列表，按字母排序返回。
// 无法访问集群或资源类型未注册到 scheme 时返回错误。
func getStorageClasses(ctx context.Context, cli sigs.Client) ([]string, error) {
	var scList storagev1.StorageClassList
	if err := cli.List(ctx, &scList); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(scList.Items))
	for _, sc := range scList.Items {
		names = append(names, sc.Name)
	}
	sort.Strings(names)
	return names, nil
}

// getAppVersions extracts the supported app version list from a package Metadata.
// Deprecated versions are excluded.
//
// getAppVersions 从包 Metadata 中提取支持的应用版本列表，排除已废弃版本。
func getAppVersions(meta *packages.Metadata) []string {
	if meta == nil {
		return nil
	}
	return meta.App.Version
}

// requiredIfNeeded returns a validation function that rejects blank input when
// required is true.
//
// requiredIfNeeded 返回一个验证函数：当 required 为 true 时拒绝空白输入。
func requiredIfNeeded(required bool) func(string) error {
	return func(s string) error {
		if required && strings.TrimSpace(s) == "" {
			return errors.New(lang.T("该字段为必填项", "this field is required"))
		}
		return nil
	}
}
