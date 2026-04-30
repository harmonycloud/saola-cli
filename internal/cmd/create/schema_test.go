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

package create

import (
	"encoding/json"
	"testing"
)

// helpers ----------------------------------------------------------------

// mustMarshal serializes v to a JSON string or panics.
//
// mustMarshal 将 v 序列化为 JSON 字符串，失败时 panic。
func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// fieldByPath returns the first FieldSchema whose Path equals target, or nil.
//
// fieldByPath 返回 Path 等于 target 的第一个 FieldSchema，未找到时返回 nil。
func fieldByPath(schemas []*FieldSchema, target string) *FieldSchema {
	for _, s := range schemas {
		if s.Path == target {
			return s
		}
	}
	return nil
}

// TestParseNecessarySchema_Basic verifies that simple top-level string/int/enum
// leaf nodes are parsed correctly.
//
// TestParseNecessarySchema_Basic 验证简单的顶级 string/int/enum 叶子节点被正确解析。
func TestParseNecessarySchema_Basic(t *testing.T) {
	raw := map[string]interface{}{
		"version": mustMarshal(map[string]interface{}{
			"type":     "version",
			"label":    "version",
			"required": true,
			"default":  "14.7",
		}),
		"locale": mustMarshal(map[string]interface{}{
			"type":    "enum",
			"label":   "Locale",
			"options": "en_US.UTF-8,zh_CN.UTF-8",
			"default": "zh_CN.UTF-8",
		}),
		"replicas": mustMarshal(map[string]interface{}{
			"type":     "int",
			"label":    "Replicas",
			"required": true,
			"default":  2,
		}),
	}

	schemas := ParseNecessarySchema(raw)
	if len(schemas) != 3 {
		t.Fatalf("expected 3 schemas, got %d", len(schemas))
	}

	ver := fieldByPath(schemas, "version")
	if ver == nil {
		t.Fatal("schema for 'version' not found")
	}
	if ver.Type != "version" {
		t.Errorf("version.Type = %q, want %q", ver.Type, "version")
	}
	if ver.Default != "14.7" {
		t.Errorf("version.Default = %v, want %q", ver.Default, "14.7")
	}
	if !ver.Required {
		t.Error("version.Required should be true")
	}

	locale := fieldByPath(schemas, "locale")
	if locale == nil {
		t.Fatal("schema for 'locale' not found")
	}
	if locale.Type != "enum" {
		t.Errorf("locale.Type = %q, want %q", locale.Type, "enum")
	}
	if locale.Options != "en_US.UTF-8,zh_CN.UTF-8" {
		t.Errorf("locale.Options = %q, want %q", locale.Options, "en_US.UTF-8,zh_CN.UTF-8")
	}

	rep := fieldByPath(schemas, "replicas")
	if rep == nil {
		t.Fatal("schema for 'replicas' not found")
	}
	if rep.Type != "int" {
		t.Errorf("replicas.Type = %q, want %q", rep.Type, "int")
	}
}

// TestParseNecessarySchema_Nested verifies that deeply nested keys produce
// the correct dot-separated Path values.
//
// TestParseNecessarySchema_Nested 验证深度嵌套的 key 产生正确的点分隔 Path。
func TestParseNecessarySchema_Nested(t *testing.T) {
	raw := map[string]interface{}{
		"resource": map[string]interface{}{
			"postgresql": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu": mustMarshal(map[string]interface{}{
						"type":     "string",
						"label":    "CPU Limit",
						"required": true,
						"min":      0.1,
					}),
					"memory": mustMarshal(map[string]interface{}{
						"type":     "string",
						"label":    "Memory Limit",
						"required": true,
					}),
				},
				"replicas": mustMarshal(map[string]interface{}{
					"type":     "int",
					"label":    "Replicas",
					"required": true,
					"default":  2,
					"pattern":  "^[2-4]$",
				}),
			},
		},
	}

	schemas := ParseNecessarySchema(raw)
	if len(schemas) != 3 {
		t.Fatalf("expected 3 schemas, got %d", len(schemas))
	}

	cpu := fieldByPath(schemas, "resource.postgresql.limits.cpu")
	if cpu == nil {
		t.Fatal("schema for 'resource.postgresql.limits.cpu' not found")
	}
	if cpu.Type != "string" {
		t.Errorf("cpu.Type = %q, want %q", cpu.Type, "string")
	}
	if cpu.Min != 0.1 {
		t.Errorf("cpu.Min = %v, want 0.1", cpu.Min)
	}

	mem := fieldByPath(schemas, "resource.postgresql.limits.memory")
	if mem == nil {
		t.Fatal("schema for 'resource.postgresql.limits.memory' not found")
	}

	rep := fieldByPath(schemas, "resource.postgresql.replicas")
	if rep == nil {
		t.Fatal("schema for 'resource.postgresql.replicas' not found")
	}
	if rep.Pattern != "^[2-4]$" {
		t.Errorf("replicas.Pattern = %q, want %q", rep.Pattern, "^[2-4]$")
	}
}

// TestParseNecessarySchema_SkipEmpty verifies that empty string leaf nodes
// are silently ignored (e.g. the "repository" field).
//
// TestParseNecessarySchema_SkipEmpty 验证空字符串叶子节点被静默忽略（如 "repository" 字段）。
func TestParseNecessarySchema_SkipEmpty(t *testing.T) {
	raw := map[string]interface{}{
		"repository": "",
		"version": mustMarshal(map[string]interface{}{
			"type":  "version",
			"label": "version",
		}),
	}

	schemas := ParseNecessarySchema(raw)
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema (empty string skipped), got %d", len(schemas))
	}
	if fieldByPath(schemas, "repository") != nil {
		t.Error("'repository' should have been skipped")
	}
}

// TestParseNecessarySchema_SkipInvalidJSON verifies that leaf strings that are
// not valid JSON objects are silently ignored.
//
// TestParseNecessarySchema_SkipInvalidJSON 验证非合法 JSON 的叶子字符串被静默忽略。
func TestParseNecessarySchema_SkipInvalidJSON(t *testing.T) {
	raw := map[string]interface{}{
		"notjson": "just a plain string",
		"partial": `{"type":`,
		"version": mustMarshal(map[string]interface{}{
			"type":  "version",
			"label": "version",
		}),
	}

	schemas := ParseNecessarySchema(raw)
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	if fieldByPath(schemas, "notjson") != nil {
		t.Error("'notjson' should have been skipped")
	}
	if fieldByPath(schemas, "partial") != nil {
		t.Error("'partial' should have been skipped")
	}
}

// TestParseNecessarySchema_Password verifies that the password type with a
// Patterns array is parsed correctly.
//
// TestParseNecessarySchema_Password 验证包含 Patterns 数组的 password 类型被正确解析。
func TestParseNecessarySchema_Password(t *testing.T) {
	raw := map[string]interface{}{
		"password": mustMarshal(map[string]interface{}{
			"type":     "password",
			"label":    "Administrator password",
			"required": true,
			"patterns": []map[string]interface{}{
				{"pattern": "^.{8,32}$", "description": "长度8-32"},
				{"pattern": "^[^\\s]+$", "description": "不能包含空格"},
			},
		}),
	}

	schemas := ParseNecessarySchema(raw)
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	pw := schemas[0]
	if pw.Type != "password" {
		t.Errorf("Type = %q, want %q", pw.Type, "password")
	}
	if !pw.Required {
		t.Error("Required should be true")
	}
	if len(pw.Patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(pw.Patterns))
	}
	if pw.Patterns[0].Pattern != "^.{8,32}$" {
		t.Errorf("Patterns[0].Pattern = %q, want %q", pw.Patterns[0].Pattern, "^.{8,32}$")
	}
	if pw.Patterns[0].Description != "长度8-32" {
		t.Errorf("Patterns[0].Description = %q, want %q", pw.Patterns[0].Description, "长度8-32")
	}
}

// TestParseNecessarySchema_SortedOutput verifies that the returned slice is
// sorted alphabetically by Path.
//
// TestParseNecessarySchema_SortedOutput 验证返回切片按 Path 字母顺序排序。
func TestParseNecessarySchema_SortedOutput(t *testing.T) {
	raw := map[string]interface{}{
		"zoo":    mustMarshal(map[string]interface{}{"type": "string", "label": "Z"}),
		"alpha":  mustMarshal(map[string]interface{}{"type": "string", "label": "A"}),
		"middle": mustMarshal(map[string]interface{}{"type": "string", "label": "M"}),
	}

	schemas := ParseNecessarySchema(raw)
	if len(schemas) != 3 {
		t.Fatalf("expected 3, got %d", len(schemas))
	}

	expected := []string{"alpha", "middle", "zoo"}
	for i, want := range expected {
		if schemas[i].Path != want {
			t.Errorf("schemas[%d].Path = %q, want %q", i, schemas[i].Path, want)
		}
	}
}

// TestBuildNecessaryValues_Flat verifies that single-segment paths are placed
// at the root level of the result map.
//
// TestBuildNecessaryValues_Flat 验证单段路径被放置在结果 map 的根层级。
func TestBuildNecessaryValues_Flat(t *testing.T) {
	flat := map[string]interface{}{
		"version": "14.7",
		"locale":  "zh_CN.UTF-8",
	}

	result := BuildNecessaryValues(flat)
	if result["version"] != "14.7" {
		t.Errorf("version = %v, want %q", result["version"], "14.7")
	}
	if result["locale"] != "zh_CN.UTF-8" {
		t.Errorf("locale = %v, want %q", result["locale"], "zh_CN.UTF-8")
	}
}

// TestBuildNecessaryValues_Nested verifies that multi-segment paths produce
// the correct nested map structure.
//
// TestBuildNecessaryValues_Nested 验证多段路径产生正确的嵌套 map 结构。
func TestBuildNecessaryValues_Nested(t *testing.T) {
	flat := map[string]interface{}{
		"resource.postgresql.limits.cpu":    "1",
		"resource.postgresql.limits.memory": "2Gi",
		"resource.postgresql.replicas":      "2",
	}

	result := BuildNecessaryValues(flat)

	resource, ok := result["resource"].(map[string]interface{})
	if !ok {
		t.Fatal("result[\"resource\"] is not a map")
	}
	pg, ok := resource["postgresql"].(map[string]interface{})
	if !ok {
		t.Fatal("resource[\"postgresql\"] is not a map")
	}
	limits, ok := pg["limits"].(map[string]interface{})
	if !ok {
		t.Fatal("postgresql[\"limits\"] is not a map")
	}
	if limits["cpu"] != "1" {
		t.Errorf("limits.cpu = %v, want %q", limits["cpu"], "1")
	}
	if limits["memory"] != "2Gi" {
		t.Errorf("limits.memory = %v, want %q", limits["memory"], "2Gi")
	}
	if pg["replicas"] != "2" {
		t.Errorf("postgresql.replicas = %v, want %q", pg["replicas"], "2")
	}
}

// TestBuildNecessaryValues_RoundTrip verifies that parsing a necessary map and
// then rebuilding values from user input produces the correct nested structure.
//
// TestBuildNecessaryValues_RoundTrip 验证解析 necessary map 后，
// 用用户输入重新构建值能产生正确的嵌套结构。
func TestBuildNecessaryValues_RoundTrip(t *testing.T) {
	// Step 1: define a representative necessary map.
	//
	// 第一步：定义一个典型的 necessary map。
	necessary := map[string]interface{}{
		"version": mustMarshal(map[string]interface{}{
			"type":     "version",
			"label":    "version",
			"required": true,
			"default":  "14.7",
		}),
		"repository": "",
		"resource": map[string]interface{}{
			"postgresql": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu": mustMarshal(map[string]interface{}{
						"type":     "string",
						"label":    "CPU Limit",
						"required": true,
					}),
					"memory": mustMarshal(map[string]interface{}{
						"type":     "string",
						"label":    "Memory Limit",
						"required": true,
					}),
				},
			},
		},
	}

	// Step 2: parse schemas to discover all required fields.
	//
	// 第二步：解析 schema 以发现所有必填字段。
	schemas := ParseNecessarySchema(necessary)

	// Expect: version, resource.postgresql.limits.cpu, resource.postgresql.limits.memory
	// "repository" must be absent.
	//
	// 预期：version、resource.postgresql.limits.cpu、resource.postgresql.limits.memory。
	// "repository" 必须缺失。
	if len(schemas) != 3 {
		t.Fatalf("expected 3 schemas, got %d", len(schemas))
	}
	for _, s := range schemas {
		if s.Path == "repository" {
			t.Error("'repository' should not appear in parsed schemas")
		}
	}

	// Step 3: simulate user filling in values (using defaults where available).
	//
	// 第三步：模拟用户填写值（有默认值时使用默认值）。
	userInput := make(map[string]interface{})
	for _, s := range schemas {
		if s.Default != nil {
			userInput[s.Path] = s.Default
		} else {
			userInput[s.Path] = "test-value"
		}
	}

	// Step 4: rebuild nested necessary values.
	//
	// 第四步：重新构建嵌套的 necessary 值。
	built := BuildNecessaryValues(userInput)

	// Verify top-level version.
	//
	// 验证顶级 version。
	if built["version"] != "14.7" {
		t.Errorf("version = %v, want %q", built["version"], "14.7")
	}

	// Verify nested cpu and memory.
	//
	// 验证嵌套的 cpu 和 memory。
	resource, ok := built["resource"].(map[string]interface{})
	if !ok {
		t.Fatal("built[\"resource\"] is not a map")
	}
	pg, ok := resource["postgresql"].(map[string]interface{})
	if !ok {
		t.Fatal("resource[\"postgresql\"] is not a map")
	}
	limits, ok := pg["limits"].(map[string]interface{})
	if !ok {
		t.Fatal("postgresql[\"limits\"] is not a map")
	}
	if limits["cpu"] != "test-value" {
		t.Errorf("limits.cpu = %v, want %q", limits["cpu"], "test-value")
	}
	if limits["memory"] != "test-value" {
		t.Errorf("limits.memory = %v, want %q", limits["memory"], "test-value")
	}
}
