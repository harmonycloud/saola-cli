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

package k8s

import (
	"context"
	"fmt"
	"testing"

	zeusv1 "github.com/opensaola/opensaola/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := zeusv1.AddToScheme(s); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	return s
}

func newFakeClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(objs...).
		Build()
}

func TestCreateMiddleware_Success(t *testing.T) {
	t.Parallel()
	cli := newFakeClient(t)
	m := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mysql",
			Namespace: "default",
		},
	}

	if err := CreateMiddleware(context.Background(), cli, m); err != nil {
		t.Fatalf("CreateMiddleware returned error: %v", err)
	}

	// Verify it was created.
	got, err := GetMiddleware(context.Background(), cli, "test-mysql", "default")
	if err != nil {
		t.Fatalf("GetMiddleware after create: %v", err)
	}
	if got.Name != "test-mysql" {
		t.Errorf("name = %q, want %q", got.Name, "test-mysql")
	}
}

func TestCreateMiddleware_AlreadyExists(t *testing.T) {
	t.Parallel()
	existing := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mysql",
			Namespace: "default",
		},
	}
	cli := newFakeClient(t, existing)

	m := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mysql",
			Namespace: "default",
		},
	}

	err := CreateMiddleware(context.Background(), cli, m)
	if err == nil {
		t.Fatal("expected AlreadyExists error, got nil")
	}
	if !errors.IsAlreadyExists(err) {
		t.Errorf("expected AlreadyExists, got: %v", err)
	}
}

func TestCreateMiddleware_GetError(t *testing.T) {
	t.Parallel()
	// Use an interceptor to make Get return a non-NotFound error.
	cli := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithInterceptorFuncs(fakeInterceptorGetError()).
		Build()

	m := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mysql",
			Namespace: "default",
		},
	}

	err := CreateMiddleware(context.Background(), cli, m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.IsNotFound(err) || errors.IsAlreadyExists(err) {
		t.Errorf("expected non-NotFound/non-AlreadyExists error, got: %v", err)
	}
}

func TestGetMiddleware_Success(t *testing.T) {
	t.Parallel()
	existing := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-redis",
			Namespace: "ns1",
		},
	}
	cli := newFakeClient(t, existing)

	got, err := GetMiddleware(context.Background(), cli, "test-redis", "ns1")
	if err != nil {
		t.Fatalf("GetMiddleware returned error: %v", err)
	}
	if got.Name != "test-redis" {
		t.Errorf("name = %q, want %q", got.Name, "test-redis")
	}
}

func TestGetMiddleware_NotFound(t *testing.T) {
	t.Parallel()
	cli := newFakeClient(t)

	_, err := GetMiddleware(context.Background(), cli, "nonexistent", "default")
	if err == nil {
		t.Fatal("expected NotFound error, got nil")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected NotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteMiddleware tests
// DeleteMiddleware 测试
// ---------------------------------------------------------------------------

// TestDeleteMiddleware_Success verifies that an existing Middleware is deleted without error.
//
// TestDeleteMiddleware_Success 验证删除已存在的 Middleware 不返回错误。
func TestDeleteMiddleware_Success(t *testing.T) {
	t.Parallel()
	existing := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mysql",
			Namespace: "default",
		},
	}
	cli := newFakeClient(t, existing)

	if err := DeleteMiddleware(context.Background(), cli, existing); err != nil {
		t.Fatalf("DeleteMiddleware returned error: %v", err)
	}

	// Verify it was deleted.
	_, err := GetMiddleware(context.Background(), cli, "test-mysql", "default")
	if err == nil {
		t.Fatal("expected NotFound after delete, got nil")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected NotFound, got: %v", err)
	}
}

// TestDeleteMiddleware_NotFound verifies that deleting a non-existent Middleware returns a NotFound error.
//
// TestDeleteMiddleware_NotFound 验证删除不存在的 Middleware 返回 NotFound 错误。
func TestDeleteMiddleware_NotFound(t *testing.T) {
	t.Parallel()
	cli := newFakeClient(t)

	m := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nonexistent",
			Namespace: "default",
		},
	}

	err := DeleteMiddleware(context.Background(), cli, m)
	if err == nil {
		t.Fatal("expected error for deleting non-existent middleware, got nil")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected NotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListMiddlewares tests
// ListMiddlewares 测试
// ---------------------------------------------------------------------------

// TestListMiddlewares_Success verifies that listing returns all Middleware CRs in a namespace.
//
// TestListMiddlewares_Success 验证列出指定命名空间下所有 Middleware CR。
func TestListMiddlewares_Success(t *testing.T) {
	t.Parallel()
	mw1 := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-1",
			Namespace: "default",
		},
	}
	mw2 := &zeusv1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-2",
			Namespace: "default",
		},
	}
	cli := newFakeClient(t, mw1, mw2)

	items, err := ListMiddlewares(context.Background(), cli, "default", nil)
	if err != nil {
		t.Fatalf("ListMiddlewares returned error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

// TestListMiddlewares_Empty verifies that listing an empty namespace returns zero items.
//
// TestListMiddlewares_Empty 验证空命名空间下列出结果为空。
func TestListMiddlewares_Empty(t *testing.T) {
	t.Parallel()
	cli := newFakeClient(t)

	items, err := ListMiddlewares(context.Background(), cli, "default", nil)
	if err != nil {
		t.Fatalf("ListMiddlewares returned error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// ---------------------------------------------------------------------------
// CreateMiddlewareOperator tests
// CreateMiddlewareOperator 测试
// ---------------------------------------------------------------------------

// TestCreateMiddlewareOperator_Success verifies that a new MiddlewareOperator is created.
//
// TestCreateMiddlewareOperator_Success 验证新建 MiddlewareOperator 成功。
func TestCreateMiddlewareOperator_Success(t *testing.T) {
	t.Parallel()
	cli := newFakeClient(t)
	mo := &zeusv1.MiddlewareOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operator",
			Namespace: "default",
		},
	}

	if err := CreateMiddlewareOperator(context.Background(), cli, mo); err != nil {
		t.Fatalf("CreateMiddlewareOperator returned error: %v", err)
	}

	// Verify it was created.
	got, err := GetMiddlewareOperator(context.Background(), cli, "test-operator", "default")
	if err != nil {
		t.Fatalf("GetMiddlewareOperator after create: %v", err)
	}
	if got.Name != "test-operator" {
		t.Errorf("name = %q, want %q", got.Name, "test-operator")
	}
}

// TestCreateMiddlewareOperator_AlreadyExists verifies that creating a duplicate
// MiddlewareOperator returns AlreadyExists.
//
// TestCreateMiddlewareOperator_AlreadyExists 验证重复创建 MiddlewareOperator 返回 AlreadyExists。
func TestCreateMiddlewareOperator_AlreadyExists(t *testing.T) {
	t.Parallel()
	existing := &zeusv1.MiddlewareOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operator",
			Namespace: "default",
		},
	}
	cli := newFakeClient(t, existing)

	mo := &zeusv1.MiddlewareOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operator",
			Namespace: "default",
		},
	}

	err := CreateMiddlewareOperator(context.Background(), cli, mo)
	if err == nil {
		t.Fatal("expected AlreadyExists error, got nil")
	}
	if !errors.IsAlreadyExists(err) {
		t.Errorf("expected AlreadyExists, got: %v", err)
	}
}

// fakeInterceptorGetError returns interceptor funcs that make Get always fail with
// a generic internal error (not NotFound).
func fakeInterceptorGetError() interceptor.Funcs {
	return interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return fmt.Errorf("simulated internal server error")
		},
	}
}
