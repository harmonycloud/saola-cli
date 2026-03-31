package main

import (
	"fmt"
	"os"

	"gitea.com/middleware-management/saola-cli/internal/app"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	// zeus-operator's package init() registers prometheus metrics against
	// prometheus.DefaultRegisterer. Replace it with a no-op registry early so
	// that the saola CLI binary does not expose (or panic on duplicate) metrics.
	//
	// zeus-operator 的包 init() 会向 prometheus.DefaultRegisterer 注册指标，
	// 提前替换为 no-op registry，避免 CLI 二进制暴露或因重复注册而 panic。
	noopRegistry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = noopRegistry
	prometheus.DefaultGatherer = noopRegistry
}

func main() {
	if err := app.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
