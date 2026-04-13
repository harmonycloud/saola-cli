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

package printer

// Package printer — name.go implements the "-o name" output format.
//
// name.go 实现 "-o name" 输出格式。

import (
	"fmt"
	"io"
	"reflect"
)

// NamePrinter outputs resources in "type/name" format, one per line.
// It mirrors the behaviour of `kubectl get -o name`.
//
// NamePrinter 以 "type/name" 格式输出资源，每行一条。
// 与 `kubectl get -o name` 的行为保持一致。
type NamePrinter struct {
	// ResourceType is the resource kind prefix (e.g. "middleware", "operator").
	// If empty, only the name is printed without a slash prefix.
	//
	// ResourceType 是资源类型前缀（如 "middleware"、"operator"）。
	// 为空时仅输出名称，不加斜杠前缀。
	ResourceType string
}

// Print writes each item's name to w in "type/name" format.
// data must be a slice of structs that have a string field named "Name" or "NAME".
//
// Print 将每个元素的名称以 "type/name" 格式写入 w。
// data 必须是包含 "Name" 或 "NAME" 字符串字段的结构体切片。
func (p *NamePrinter) Print(w io.Writer, data interface{}) error {
	v := reflect.ValueOf(data)

	// Dereference pointer if needed.
	//
	// 必要时解引用指针。
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Accept both slice and a single struct.
	//
	// 同时支持切片和单个结构体。
	var items []reflect.Value
	switch v.Kind() {
	case reflect.Slice:
		items = make([]reflect.Value, v.Len())
		for i := range items {
			items[i] = v.Index(i)
		}
	case reflect.Struct:
		items = []reflect.Value{v}
	default:
		return fmt.Errorf("NamePrinter: unsupported data type %T", data)
	}

	for _, item := range items {
		name, err := extractName(item)
		if err != nil {
			return err
		}
		if p.ResourceType != "" {
			fmt.Fprintf(w, "%s/%s\n", p.ResourceType, name)
		} else {
			fmt.Fprintf(w, "%s\n", name)
		}
	}
	return nil
}

// extractName looks up a "Name" or "NAME" string field from a struct value.
// It also handles struct values wrapped in an interface or pointer.
//
// extractName 从结构体值中查找 "Name" 或 "NAME" 字符串字段。
// 也处理通过 interface 或指针包裹的结构体。
func extractName(v reflect.Value) (string, error) {
	// Unwrap interface / pointer.
	//
	// 解包 interface / 指针。
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("NamePrinter: expected struct, got %s", v.Kind())
	}

	// Try "Name" then "NAME".
	//
	// 依次尝试 "Name" 和 "NAME" 字段。
	for _, fieldName := range []string{"Name", "NAME"} {
		f := v.FieldByName(fieldName)
		if f.IsValid() && f.Kind() == reflect.String {
			return f.String(), nil
		}
	}

	// Fall back to embedded ObjectMeta.Name via nested struct traversal.
	//
	// 回退：从嵌入的 ObjectMeta 中递归查找 Name 字段。
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		if ft.Anonymous && ft.Type.Kind() == reflect.Struct {
			inner := v.Field(i)
			name, err := extractName(inner)
			if err == nil {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("NamePrinter: struct %s has no Name or NAME field", v.Type().Name())
}
