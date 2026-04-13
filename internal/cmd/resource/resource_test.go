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

package resource

import (
	"testing"
)

// TestResolve verifies alias resolution and passthrough for known and unknown inputs.
//
// TestResolve 验证已知别名解析和未知输入的直通行为。
func TestResolve(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"mw", "middleware"},
		{"op", "operator"},
		{"act", "action"},
		{"bl", "baseline"},
		{"pkg", "package"},
		{"middleware", "middleware"}, // passthrough — canonical name unchanged
		{"operator", "operator"},    // passthrough
		{"action", "action"},        // passthrough
		{"foo", "foo"},              // unknown passthrough
		{"", ""},                    // empty passthrough
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := Resolve(tt.input)
			if got != tt.expected {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
