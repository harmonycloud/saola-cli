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

	"github.com/harmonycloud/saola-cli/internal/client"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/packages"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOptions holds parameters for lightweight "get package <name>".
// GetOptions 保存轻量级 "get package <name>" 的参数。
type GetOptions struct {
	Config *config.Config
	Name   string
	Output string
	// Client is the k8s client to use. If nil, a real client is built from Config.
	//
	// Client 为注入的 k8s 客户端；为 nil 时从 Config 构建真实客户端（供测试注入 fake client）。
	Client sigs.Client
}

func (o *GetOptions) Run(ctx context.Context) error {
	if o.Name == "" {
		return fmt.Errorf("package name is required")
	}
	packages.SetDataNamespace(o.Config.PkgNamespace)

	cli := o.Client
	if cli == nil {
		var err error
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	pkg, err := packages.GetSummary(ctx, cli, o.Name)
	if err != nil {
		return fmt.Errorf("get package metadata: %w", err)
	}
	return printPackage(o.Output, pkg)
}
