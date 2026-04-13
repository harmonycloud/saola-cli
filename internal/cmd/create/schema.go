package create

import (
	"encoding/json"
	"sort"
	"strings"
)

// FieldSchema represents a single field's JSON Schema from baseline necessary.
//
// FieldSchema 表示 baseline necessary 中一个字段的 JSON Schema 定义。
type FieldSchema struct {
	// Path is the dot-separated path to this field within the necessary map, e.g. "resource.postgresql.limits.cpu".
	// It is populated by ParseNecessarySchema and is not part of the JSON payload.
	//
	// Path 是该字段在 necessary map 中的点分隔路径，如 "resource.postgresql.limits.cpu"。
	// 由 ParseNecessarySchema 填充，不属于 JSON 载荷。
	Path string `json:"-"`

	// Type identifies the field kind: string | int | password | enum | version | storageClass.
	//
	// Type 标识字段类型：string | int | password | enum | version | storageClass。
	Type string `json:"type"`

	// Label is the human-readable display name shown in UI or prompts.
	//
	// Label 是在 UI 或交互提示中展示的人类可读名称。
	Label string `json:"label"`

	// Required indicates whether the user must provide a value for this field.
	//
	// Required 表示该字段是否为必填项。
	Required bool `json:"required"`

	// Default is the pre-filled value when the user provides no input.
	// It is typed as interface{} to accommodate strings, numbers, and booleans.
	//
	// Default 是用户未填写时的预填值，类型为 interface{} 以兼容字符串、数字和布尔值。
	Default interface{} `json:"default"`

	// Description provides additional context or help text for the field.
	//
	// Description 提供字段的补充说明或帮助文本。
	Description string `json:"description"`

	// Placeholder is the hint text displayed inside an empty input box.
	//
	// Placeholder 是空输入框内的提示文本。
	Placeholder string `json:"placeholder"`

	// Pattern is a single regex pattern used for simple input validation.
	//
	// Pattern 是用于简单输入验证的单个正则表达式模式。
	Pattern string `json:"pattern"`

	// Min is the lower bound for numeric fields (inclusive).
	//
	// Min 是数值字段的下界（含）。
	Min float64 `json:"min"`

	// Max is the upper bound for numeric fields (inclusive).
	//
	// Max 是数值字段的上界（含）。
	Max float64 `json:"max"`

	// Options is a comma-separated list of allowed values for enum fields.
	//
	// Options 是 enum 类型字段的逗号分隔允许值列表。
	Options string `json:"options"`

	// Patterns holds multiple validation rules, each with its own regex and description.
	// Primarily used by the password type.
	//
	// Patterns 保存多条验证规则，每条规则有独立的正则和描述，主要用于 password 类型。
	Patterns []PatternRule `json:"patterns"`
}

// PatternRule defines a validation pattern with a human-readable description.
//
// PatternRule 定义一个带人类可读描述的验证模式。
type PatternRule struct {
	// Pattern is the regex expression to match against the user's input.
	//
	// Pattern 是用于匹配用户输入的正则表达式。
	Pattern string `json:"pattern"`

	// Description explains what this pattern enforces, shown as a validation hint.
	//
	// Description 解释该模式所强制的约束，作为验证提示展示给用户。
	Description string `json:"description"`
}

// ParseNecessarySchema recursively walks the baseline necessary map and extracts
// all leaf-node JSON Schema definitions. Returns a sorted slice of FieldSchema.
//
// ParseNecessarySchema 递归遍历 baseline necessary map，提取所有叶子节点的
// JSON Schema 定义。返回按 Path 字母排序的 FieldSchema 切片。
//
// Parsing rules:
//   - A leaf node that is a string → attempt JSON unmarshal into FieldSchema.
//   - Empty string or invalid JSON → skip (covers NecessaryIgnore-style fields like "repository").
//   - A non-string map value → recurse with the current key appended to prefix.
func ParseNecessarySchema(raw map[string]interface{}) []*FieldSchema {
	var results []*FieldSchema
	walkNecessary(raw, "", &results)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})
	return results
}

// walkNecessary is the internal recursive helper for ParseNecessarySchema.
//
// walkNecessary 是 ParseNecessarySchema 的内部递归辅助函数。
func walkNecessary(node map[string]interface{}, prefix string, out *[]*FieldSchema) {
	for k, v := range node {
		// Build the dot-separated path for this key.
		//
		// 构建当前 key 的点分隔路径。
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			// Non-leaf node: recurse deeper.
			//
			// 非叶子节点：继续向下递归。
			walkNecessary(val, path, out)

		case string:
			// Leaf node: skip empty strings silently.
			//
			// 叶子节点：静默跳过空字符串。
			if val == "" {
				continue
			}

			// Attempt to unmarshal the string as a FieldSchema.
			//
			// 尝试将字符串反序列化为 FieldSchema。
			var fs FieldSchema
			if err := json.Unmarshal([]byte(val), &fs); err != nil {
				// Not valid JSON or not a schema definition — skip.
				//
				// 不是合法 JSON 或不是 schema 定义——跳过。
				continue
			}

			// A valid schema must carry a non-empty type.
			//
			// 合法的 schema 必须包含非空的 type 字段。
			if fs.Type == "" {
				continue
			}

			fs.Path = path
			*out = append(*out, &fs)

		default:
			// Other scalar types (bool, number, nil) are not schema definitions — skip.
			//
			// 其他标量类型（bool、number、nil）不是 schema 定义——跳过。
		}
	}
}

// BuildNecessaryValues converts a flat map of path→value back into a nested map
// suitable for Middleware.Spec.Necessary.
//
// BuildNecessaryValues 将 path→value 的扁平 map 转换回嵌套 map，
// 用于构建 Middleware.Spec.Necessary。
//
// Example:
//
//	{"resource.postgresql.limits.cpu": "1"} →
//	{"resource": {"postgresql": {"limits": {"cpu": "1"}}}}
func BuildNecessaryValues(flat map[string]interface{}) map[string]interface{} {
	root := make(map[string]interface{})

	for path, value := range flat {
		segments := strings.Split(path, ".")
		current := root

		// Navigate / create intermediate maps for all but the last segment.
		//
		// 对除最后一段之外的所有段，逐级导航并在必要时创建中间 map。
		for _, seg := range segments[:len(segments)-1] {
			if existing, ok := current[seg]; ok {
				// Reuse an existing nested map.
				//
				// 复用已存在的嵌套 map。
				if nested, ok := existing.(map[string]interface{}); ok {
					current = nested
					continue
				}
			}
			// Create a new intermediate map.
			//
			// 创建新的中间 map。
			nested := make(map[string]interface{})
			current[seg] = nested
			current = nested
		}

		// Set the leaf value under the last segment.
		//
		// 在最后一段下设置叶子值。
		leaf := segments[len(segments)-1]
		current[leaf] = value
	}

	return root
}
