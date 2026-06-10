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

package pkgcmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

type packageUsage struct {
	Kind      string
	Namespace string
	Name      string
	Baseline  string
	State     string
}

func findPackageUsages(ctx context.Context, cli sigs.Client, packageName string) ([]packageUsage, error) {
	selector := sigs.MatchingLabels{zeusv1.LabelPackageName: packageName}
	var usages []packageUsage

	middlewares := &zeusv1.MiddlewareList{}
	if err := cli.List(ctx, middlewares, selector); err != nil {
		return nil, fmt.Errorf("list Middleware by package %q: %w", packageName, err)
	}
	for _, item := range middlewares.Items {
		usages = append(usages, packageUsage{
			Kind:      "middleware",
			Namespace: item.Namespace,
			Name:      item.Name,
			Baseline:  item.Spec.Baseline,
			State:     string(item.Status.State),
		})
	}

	operators := &zeusv1.MiddlewareOperatorList{}
	if err := cli.List(ctx, operators, selector); err != nil {
		return nil, fmt.Errorf("list MiddlewareOperator by package %q: %w", packageName, err)
	}
	for _, item := range operators.Items {
		usages = append(usages, packageUsage{
			Kind:      "middlewareoperator",
			Namespace: item.Namespace,
			Name:      item.Name,
			Baseline:  item.Spec.Baseline,
			State:     string(item.Status.State),
		})
	}

	sort.Slice(usages, func(i, j int) bool {
		left := usages[i].Kind + "/" + usages[i].Namespace + "/" + usages[i].Name
		right := usages[j].Kind + "/" + usages[j].Namespace + "/" + usages[j].Name
		return left < right
	})
	return usages, nil
}

func packageUsageError(packageName string, usages []packageUsage) error {
	if len(usages) == 0 {
		return nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "package %q is still used by deployed Middleware or MiddlewareOperator resources; uninstall those resources before uninstalling the package:", packageName)
	for _, usage := range usages {
		fmt.Fprintf(&b, "\n- %s %s/%s", usage.Kind, usage.Namespace, usage.Name)
		if usage.Baseline != "" {
			fmt.Fprintf(&b, " (baseline=%s", usage.Baseline)
			if usage.State != "" {
				fmt.Fprintf(&b, ", state=%s", usage.State)
			}
			b.WriteString(")")
		} else if usage.State != "" {
			fmt.Fprintf(&b, " (state=%s)", usage.State)
		}
		fmt.Fprintf(&b, "; delete with: saola delete %s %s -n %s", deleteCommandResource(usage.Kind), usage.Name, usage.Namespace)
	}
	return fmt.Errorf("%s", b.String())
}

func deleteCommandResource(kind string) string {
	if kind == "middlewareoperator" {
		return "operator"
	}
	return "middleware"
}
