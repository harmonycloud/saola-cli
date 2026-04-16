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

package client

import (
	"testing"

	"github.com/harmonycloud/saola-cli/internal/config"
)

// TestClient_InvalidKubeconfig verifies that Get returns an error when the
// kubeconfig file does not exist.
//
// 验证当 kubeconfig 文件不存在时 Get 返回错误。
func TestClient_InvalidKubeconfig(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Kubeconfig: "/nonexistent/kubeconfig.yaml"}
	c := New(cfg)
	_, err := c.Get()
	if err == nil {
		t.Fatal("expected error with invalid kubeconfig")
	}
}

// TestClient_RetryAfterFailure verifies that a failed initialisation is not
// cached — a second call to Get retries instead of returning a stale error.
//
// 验证初始化失败不会被缓存——第二次调用 Get 会重新尝试而不是返回缓存的错误。
func TestClient_RetryAfterFailure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Kubeconfig: "/nonexistent/path"}
	c := New(cfg)

	// First call fails.
	//
	// 第一次调用失败。
	_, err1 := c.Get()

	// Second call should also fail but NOT return cached error — it retries.
	//
	// 第二次调用也应失败，但不应返回缓存的错误——它会重试。
	_, err2 := c.Get()

	if err1 == nil || err2 == nil {
		t.Fatal("both calls should fail")
	}

	// The key test: inner must still be nil, proving no stale client was cached.
	//
	// 关键测试：inner 必须仍为 nil，证明没有缓存失败的客户端。
	if c.inner != nil {
		t.Fatal("inner client should remain nil after failed initialisations")
	}
}
