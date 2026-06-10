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
	"os"

	zeusv1 "github.com/harmonycloud/opensaola/api/v1"
	"github.com/harmonycloud/saola-cli/internal/client"
	"github.com/harmonycloud/saola-cli/internal/config"
	"github.com/harmonycloud/saola-cli/internal/lang"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// DisableOptions holds parameters for disabling a package without deleting it.
//
// DisableOptions 保存禁用包的参数；禁用不会删除包 Secret。
type DisableOptions struct {
	Config *config.Config
	Name   string
	Client sigs.Client
}

// NewCmdDisable returns the "package disable" command.
//
// NewCmdDisable 返回 package disable 子命令。
func NewCmdDisable(cfg *config.Config) *cobra.Command {
	o := &DisableOptions{Config: cfg}
	cmd := &cobra.Command{
		Use:   "disable <name>",
		Short: lang.T("禁用中间件包", "Disable a middleware package"),
		Long: lang.T(
			`将包 Secret 标记为 disabled，不删除 Secret，也不清理已发布的包资源。`,
			`Mark the package Secret as disabled without deleting the Secret or cleaning published package resources.`,
		),
		Example: `  saola package disable redis-v1`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}
	return cmd
}

// NewCmdDisablePackage returns the "disable package" command.
//
// NewCmdDisablePackage 返回 disable package 子命令。
func NewCmdDisablePackage(cfg *config.Config) *cobra.Command {
	o := &DisableOptions{Config: cfg}
	cmd := &cobra.Command{
		Use:     "package <name>",
		Aliases: []string{"pkg"},
		Short:   lang.T("禁用中间件包", "Disable a middleware package"),
		Long: lang.T(
			`将包 Secret 标记为 disabled，不删除 Secret，也不清理已发布的包资源。`,
			`Mark the package Secret as disabled without deleting the Secret or cleaning published package resources.`,
		),
		Example: `  saola disable package redis-v1`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}
	return cmd
}

func (o *DisableOptions) Run(ctx context.Context) error {
	var err error
	cli := o.Client
	if cli == nil {
		cli, err = client.New(o.Config).Get()
		if err != nil {
			return fmt.Errorf("create k8s client: %w", err)
		}
	}

	secret := &corev1.Secret{}
	if err = cli.Get(ctx, sigs.ObjectKey{Name: o.Name, Namespace: o.Config.PkgNamespace}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("package Secret %q not found in namespace %q", o.Name, o.Config.PkgNamespace)
		}
		return fmt.Errorf("get Secret: %w", err)
	}
	if _, ok := secret.Annotations[zeusv1.LabelInstall]; ok {
		return fmt.Errorf("package %q has a pending install; wait for it to finish before disabling", o.Name)
	}
	if _, ok := secret.Annotations[zeusv1.LabelUnInstall]; ok {
		return fmt.Errorf("package %q has a pending uninstall; wait for it to finish before disabling", o.Name)
	}

	patch := sigs.MergeFrom(secret.DeepCopy())
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	secret.Labels[zeusv1.LabelEnabled] = "false"
	if err = cli.Patch(ctx, secret, patch); err != nil {
		return fmt.Errorf("patch Secret: %w", err)
	}

	mp := &zeusv1.MiddlewarePackage{}
	if err = cli.Get(ctx, sigs.ObjectKey{Name: o.Name}, mp); err == nil {
		mpPatch := sigs.MergeFrom(mp.DeepCopy())
		if mp.Labels == nil {
			mp.Labels = make(map[string]string)
		}
		mp.Labels[zeusv1.LabelEnabled] = "false"
		if err = cli.Patch(ctx, mp, mpPatch); err != nil {
			return fmt.Errorf("patch MiddlewarePackage: %w", err)
		}
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get MiddlewarePackage: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Package %s disabled\n", o.Name)
	return nil
}
