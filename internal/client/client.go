package client

import (
	"fmt"
	"sync"

	zeusv1 "gitee.com/opensaola/opensaola/api/v1"
	"gitee.com/opensaola/saola-cli/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps a controller-runtime client with lazy initialisation.
//
// Client 封装了一个延迟初始化的 controller-runtime 客户端。
type Client struct {
	cfg    *config.Config
	once   sync.Once
	inner  client.Client
	initErr error
}

// New creates a lazily-initialised Client.
// The actual connection to the cluster is deferred until the first call to Get().
//
// 创建一个延迟初始化的 Client，实际集群连接在首次调用 Get() 时建立。
func New(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

// Get returns the underlying controller-runtime client, initialising it on first call.
//
// 返回底层 controller-runtime 客户端，首次调用时完成初始化。
func (c *Client) Get() (client.Client, error) {
	c.once.Do(func() {
		restCfg, err := c.buildRestConfig()
		if err != nil {
			c.initErr = fmt.Errorf("build rest config: %w", err)
			return
		}

		scheme := runtime.NewScheme()
		if err = corev1.AddToScheme(scheme); err != nil {
			c.initErr = fmt.Errorf("add corev1 scheme: %w", err)
			return
		}
		if err = appsv1.AddToScheme(scheme); err != nil {
			c.initErr = fmt.Errorf("add appsv1 scheme: %w", err)
			return
		}
		if err = storagev1.AddToScheme(scheme); err != nil {
			c.initErr = fmt.Errorf("add storage/v1 scheme: %w", err)
			return
		}
		if err = zeusv1.AddToScheme(scheme); err != nil {
			c.initErr = fmt.Errorf("add zeus/v1 scheme: %w", err)
			return
		}

		c.inner, c.initErr = client.New(restCfg, client.Options{Scheme: scheme})
	})
	return c.inner, c.initErr
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
