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

package waiter

import (
	"context"
	"fmt"
	"time"

	zeusv1 "github.com/opensaola/opensaola/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var pollInterval = 2 * time.Second

// WaitForInstall polls until the package Secret becomes enabled=true (success),
// an installError annotation is set (failure), or the context deadline is exceeded.
//
// 轮询直到包 Secret 变为 enabled=true（成功）、installError 注解被设置（失败），或超过上下文截止时间。
func WaitForInstall(ctx context.Context, cli client.Client, name, namespace string) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for package %q to install: %w", name, ctx.Err())
		case <-time.After(pollInterval):
		}

		secret := &corev1.Secret{}
		if err := cli.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
			return fmt.Errorf("get secret %q: %w", name, err)
		}

		if secret.Labels[zeusv1.LabelEnabled] == "true" {
			return nil
		}

		if secret.Annotations != nil {
			if errMsg := secret.Annotations[zeusv1.AnnotationInstallError]; errMsg != "" {
				return fmt.Errorf("package %q install failed: %s", name, errMsg)
			}
		}
	}
}

// WaitForUninstall polls until the package Secret is deleted or no longer has the
// uninstall annotation and enabled label is "false", or until the context deadline is exceeded.
// A NotFound error from the API server is treated as successful uninstallation (Secret was deleted directly).
//
// 轮询直到包 Secret 被删除，或已清除卸载注解且 enabled=false，或超过上下文截止时间。
// API 返回 NotFound 视为卸载成功（Secret 已被直接删除）。
func WaitForUninstall(ctx context.Context, cli client.Client, name, namespace string) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for package %q to uninstall: %w", name, ctx.Err())
		case <-time.After(pollInterval):
		}

		secret := &corev1.Secret{}
		if err := cli.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
			// NotFound means the Secret was deleted directly — treat as successful uninstall.
			//
			// NotFound 表示 Secret 已被直接删除，视为卸载成功。
			if apierrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("get secret %q: %w", name, err)
		}

		enabled := secret.Labels[zeusv1.LabelEnabled] == "true"
		_, hasUninstallAnnotation := secret.Annotations[zeusv1.LabelUnInstall]

		// Uninstall is complete when the operator has cleared the annotation and
		// left enabled=false (package is dormant / removed).
		//
		// 当 operator 清除注解且 enabled=false 时，卸载完成。
		if !enabled && !hasUninstallAnnotation {
			return nil
		}
	}
}
