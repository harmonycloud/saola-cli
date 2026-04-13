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

package main

import (
	"fmt"
	"os"

	"gitee.com/opensaola/saola-cli/internal/app"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	// opensaola's package init() registers prometheus metrics against
	// prometheus.DefaultRegisterer. Replace it with a no-op registry early so
	// that the saola CLI binary does not expose (or panic on duplicate) metrics.
	//
	// opensaola 的包 init() 会向 prometheus.DefaultRegisterer 注册指标，
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
