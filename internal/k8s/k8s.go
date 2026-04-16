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

// Package k8s provides thin CRUD helpers for CRs and Secrets used by saola-cli.
// These functions were previously imported from opensaola/pkg/k8s, which has
// since moved to opensaola/internal/k8s and is no longer importable.
//
// k8s 包提供 saola-cli 所需的 CR 和 Secret 的轻量 CRUD 辅助函数。
// 这些函数原先从 opensaola/pkg/k8s 导入，现已迁移至 opensaola/internal/k8s，
// 外部模块无法再导入。
package k8s

import (
	"context"
	"fmt"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ---------------------------------------------------------------------------
// Middleware helpers
// Middleware 辅助函数
// ---------------------------------------------------------------------------

// MiddlewareGroupResource returns the GroupResource for Middleware.
//
// 返回 Middleware 的 GroupResource。
func MiddlewareGroupResource() schema.GroupResource {
	return schema.GroupResource{Group: "middleware.cn", Resource: "Middleware"}
}

// CreateMiddleware creates a Middleware CR, returning AlreadyExists if it exists.
//
// 创建 Middleware CR，若已存在则返回 AlreadyExists 错误。
func CreateMiddleware(ctx context.Context, cli client.Client, m *zeusv1.Middleware) error {
	_, err := GetMiddleware(ctx, cli, m.Name, m.Namespace)
	if err == nil {
		return errors.NewAlreadyExists(MiddlewareGroupResource(), m.Name)
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("check middleware existence: %w", err)
	}
	return cli.Create(ctx, m)
}

// GetMiddleware retrieves a single Middleware by name and namespace.
//
// 根据 name 和 namespace 获取单个 Middleware。
func GetMiddleware(ctx context.Context, cli client.Client, name, namespace string) (*zeusv1.Middleware, error) {
	m := new(zeusv1.Middleware)
	err := cli.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ListMiddlewares lists all Middleware CRs in the given namespace matching labels.
//
// 列出指定命名空间下符合 label 筛选条件的所有 Middleware CR。
func ListMiddlewares(ctx context.Context, cli client.Client, namespace string, labels client.MatchingLabels) ([]zeusv1.Middleware, error) {
	list := new(zeusv1.MiddlewareList)
	if err := cli.List(ctx, list, client.InNamespace(namespace), labels); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// DeleteMiddleware deletes a Middleware CR.
//
// 删除 Middleware CR。
func DeleteMiddleware(ctx context.Context, cli client.Client, m *zeusv1.Middleware) error {
	return cli.Delete(ctx, m)
}

// ---------------------------------------------------------------------------
// MiddlewareOperator helpers
// MiddlewareOperator 辅助函数
// ---------------------------------------------------------------------------

// MiddlewareOperatorGroupResource returns the GroupResource for MiddlewareOperator.
//
// 返回 MiddlewareOperator 的 GroupResource。
func MiddlewareOperatorGroupResource() schema.GroupResource {
	return schema.GroupResource{Group: "middleware.cn", Resource: "MiddlewareOperator"}
}

// CreateMiddlewareOperator creates a MiddlewareOperator CR, returning AlreadyExists if it exists.
//
// 创建 MiddlewareOperator CR，若已存在则返回 AlreadyExists 错误。
func CreateMiddlewareOperator(ctx context.Context, cli client.Client, m *zeusv1.MiddlewareOperator) error {
	_, err := GetMiddlewareOperator(ctx, cli, m.Name, m.Namespace)
	if err == nil {
		return errors.NewAlreadyExists(MiddlewareOperatorGroupResource(), m.Name)
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("check middleware operator existence: %w", err)
	}
	return cli.Create(ctx, m)
}

// GetMiddlewareOperator retrieves a single MiddlewareOperator by name and namespace.
//
// 根据 name 和 namespace 获取单个 MiddlewareOperator。
func GetMiddlewareOperator(ctx context.Context, cli client.Client, name, namespace string) (*zeusv1.MiddlewareOperator, error) {
	m := new(zeusv1.MiddlewareOperator)
	err := cli.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Secret helpers
// Secret 辅助函数
// ---------------------------------------------------------------------------

// GetSecret retrieves a single Secret by name and namespace.
//
// 根据 name 和 namespace 获取单个 Secret。
func GetSecret(ctx context.Context, cli client.Client, name, namespace string) (*corev1.Secret, error) {
	s := new(corev1.Secret)
	err := cli.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetSecrets lists Secrets in the given namespace matching labels.
//
// 列出指定命名空间下符合 label 筛选条件的 Secret 列表。
func GetSecrets(ctx context.Context, cli client.Client, namespace string, labelSelector client.MatchingLabels) (*corev1.SecretList, error) {
	if cli == nil {
		return nil, fmt.Errorf("k8s client is nil")
	}
	list := new(corev1.SecretList)
	if err := cli.List(ctx, list, client.InNamespace(namespace), labelSelector); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	return list, nil
}
