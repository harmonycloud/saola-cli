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

package images

import (
	"context"
	"os/exec"
)

// Runner abstracts external command execution for tests.
//
// Runner 抽象外部命令执行，方便单元测试注入。
type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args ...string) error
}

// ExecRunner executes real host commands.
//
// ExecRunner 执行真实宿主机命令。
type ExecRunner struct{}

func (r ExecRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (r ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

func defaultRunner(r Runner) Runner {
	if r != nil {
		return r
	}
	return ExecRunner{}
}
