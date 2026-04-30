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
	"fmt"
	"sync"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps a controller-runtime client with lazy initialization.
// Unlike sync.Once, failed initialization is not cached — subsequent calls to
// Get() will retry, which is useful when transient errors (e.g. network) occur.
//
// Client 封装了一个延迟初始化的 controller-runtime 客户端。
// 与 sync.Once 不同，初始化失败不会被缓存——后续调用 Get() 会重试，
// 这在发生瞬时错误（如网络问题）时非常有用。
type Client struct {
	cfg   *config.Config
	mu    sync.Mutex
	inner client.Client
}

// New creates a lazily-initialized Client.
// The actual connection to the cluster is deferred until the first call to Get().
//
// 创建一个延迟初始化的 Client，实际集群连接在首次调用 Get() 时建立。
func New(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

// Get returns the underlying controller-runtime client, initializing it on first call.
// If initialization fails, the error is returned but not cached — a subsequent call
// will attempt initialization again.
//
// 返回底层 controller-runtime 客户端，首次调用时完成初始化。
// 如果初始化失败，返回错误但不缓存——后续调用会重新尝试初始化。
func (c *Client) Get() (client.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.inner != nil {
		return c.inner, nil
	}

	restCfg, err := c.buildRestConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("add corev1 scheme: %w", err)
	}
	if err = appsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("add appsv1 scheme: %w", err)
	}
	if err = storagev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("add storage/v1 scheme: %w", err)
	}
	if err = zeusv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("add zeus/v1 scheme: %w", err)
	}

	cli, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	c.inner = cli
	return cli, nil
}

func (c *Client) buildRestConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if c.cfg.Kubeconfig != "" {
		loadingRules.ExplicitPath = c.cfg.Kubeconfig
	}

	overrides := &clientcmd.ConfigOverrides{}
	if c.cfg.Context != "" {
		overrides.CurrentContext = c.cfg.Context
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	return kubeConfig.ClientConfig()
}
