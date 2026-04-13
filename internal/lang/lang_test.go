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

package lang

import "testing"

func TestT_DefaultChinese(t *testing.T) {
	// Save and restore the current value to avoid polluting other tests.
	orig := current
	defer func() { current = orig }()

	current = "zh"
	got := T("Chinese text", "English text")
	if got != "Chinese text" {
		t.Errorf("T() with zh = %q, want %q", got, "Chinese text")
	}
}

func TestT_English(t *testing.T) {
	orig := current
	defer func() { current = orig }()

	current = "en"
	got := T("Chinese text", "English text")
	if got != "English text" {
		t.Errorf("T() with en = %q, want %q", got, "English text")
	}
}
