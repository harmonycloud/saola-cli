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

package consts

import "testing"

// TestConstantValues verifies that critical constant values remain stable
// across refactoring. These values are used in Kubernetes labels and Secret
// data keys; any accidental change would break runtime behaviour.
//
// 验证关键常量值在重构过程中保持稳定。这些值用于 Kubernetes labels 和
// Secret data keys，任何意外变更都会导致运行时行为异常。
func TestConstantValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "LabelDefinition",
			got:  LabelDefinition,
			want: "middleware.cn/definition",
		},
		{
			name: "ProjectOpenSaola",
			got:  ProjectOpenSaola,
			want: "opensaola",
		},
		{
			name: "Release",
			got:  Release,
			want: "package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}
