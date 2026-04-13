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

package version

import (
	"strings"
	"testing"
)

// Get() should return default values when package-level vars have not been changed.
//
// 未修改包级变量时，Get() 应返回默认值。
func TestGet_Defaults(t *testing.T) {
	// Restore original values after test.
	// 测试结束后恢复原始值。
	origVersion, origCommit, origDate := Version, GitCommit, BuildDate
	defer func() {
		Version = origVersion
		GitCommit = origCommit
		BuildDate = origDate
	}()

	Version = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"

	info := Get()
	if info.Version != "dev" {
		t.Errorf("Version: want %q, got %q", "dev", info.Version)
	}
	if info.GitCommit != "unknown" {
		t.Errorf("GitCommit: want %q, got %q", "unknown", info.GitCommit)
	}
	if info.BuildDate != "unknown" {
		t.Errorf("BuildDate: want %q, got %q", "unknown", info.BuildDate)
	}
}

// Get() should reflect updated package-level variables.
//
// 修改包级变量后，Get() 应返回更新后的值。
func TestGet_Modified(t *testing.T) {
	origVersion, origCommit, origDate := Version, GitCommit, BuildDate
	defer func() {
		Version = origVersion
		GitCommit = origCommit
		BuildDate = origDate
	}()

	Version = "v1.2.3"
	GitCommit = "abc123"
	BuildDate = "2026-03-30"

	info := Get()
	if info.Version != "v1.2.3" {
		t.Errorf("Version: want %q, got %q", "v1.2.3", info.Version)
	}
	if info.GitCommit != "abc123" {
		t.Errorf("GitCommit: want %q, got %q", "abc123", info.GitCommit)
	}
	if info.BuildDate != "2026-03-30" {
		t.Errorf("BuildDate: want %q, got %q", "2026-03-30", info.BuildDate)
	}
}

// Info.String() should contain all three fields.
//
// Info.String() 的输出应包含全部三个字段的值。
func TestInfo_String_ContainsAllFields(t *testing.T) {
	info := Info{
		Version:   "v2.0.0",
		GitCommit: "deadbeef",
		BuildDate: "2026-01-01",
	}
	s := info.String()
	for _, want := range []string{"v2.0.0", "deadbeef", "2026-01-01"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() missing %q in output: %q", want, s)
		}
	}
}

// Info.String() should match the expected format exactly.
//
// Info.String() 的格式应与预期完全一致。
func TestInfo_String_Format(t *testing.T) {
	info := Info{
		Version:   "v0.1.0",
		GitCommit: "ff00",
		BuildDate: "2025-12-31",
	}
	want := "Version: v0.1.0\nGit Commit: ff00\nBuild Date: 2025-12-31"
	got := info.String()
	if got != want {
		t.Errorf("String() format mismatch:\nwant: %q\ngot:  %q", want, got)
	}
}
