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
	cli := newFakeClient(t)

	_, err := GetMiddleware(context.Background(), cli, "nonexistent", "default")
	if err == nil {
		t.Fatal("expected NotFound error, got nil")
	}
	if !errors.IsNotFound(err) {
		t.Errorf("expected NotFound, got: %v", err)
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
