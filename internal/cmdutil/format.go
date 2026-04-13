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

package cmdutil

import (
	"fmt"
	"strings"
	"time"
)

// FormatAge formats a duration as a human-readable age string (e.g., "5d", "3h", "45m", "10s").
//
// FormatAge 将 duration 格式化为人类可读的 age 字符串（例如 "5d"、"3h"、"45m"、"10s"）。
func FormatAge(d time.Duration) string {
	if d < 0 {
		return "<unknown>"
	}
	if d.Hours() >= 24 {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// Truncate shortens a string to max length, appending "..." if truncated.
// The returned string never exceeds max characters.
//
// Truncate 将字符串截断到 max 个字符，超出时追加 "..."。
// 返回的字符串不会超过 max 个字符。
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// FormatLabelsShort returns a compact label summary for table display.
//
// FormatLabelsShort 返回用于表格显示的紧凑 label 摘要。
func FormatLabelsShort(labels map[string]string, keys []string) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := labels[k]; ok && v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, "/")
}
