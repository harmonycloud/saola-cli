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
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"Days", 72 * time.Hour, "3d"},
		{"Hours", 5 * time.Hour, "5h"},
		{"Minutes", 45 * time.Minute, "45m"},
		{"Seconds", 10 * time.Second, "10s"},
		{"Negative", -1 * time.Second, "<unknown>"},
		{"Zero", 0, "0s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatAge(tt.d)
			if got != tt.want {
				t.Errorf("FormatAge(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"Short", "hi", 10, "hi"},
		{"Exact", "hello", 5, "hello"},
		{"Long", "hello world", 8, "hello..."},
		{"Max3", "hello", 3, "hel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestFormatLabelsShort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		labels map[string]string
		keys   []string
		want   string
	}{
		{
			"Normal",
			map[string]string{"app": "redis", "version": "7.0"},
			[]string{"app", "version"},
			"redis/7.0",
		},
		{
			"MissingKey",
			map[string]string{"app": "redis"},
			[]string{"app", "version"},
			"redis",
		},
		{
			"Empty",
			map[string]string{},
			[]string{"app"},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatLabelsShort(tt.labels, tt.keys)
			if got != tt.want {
				t.Errorf("FormatLabelsShort() = %q, want %q", got, tt.want)
			}
		})
	}
}
