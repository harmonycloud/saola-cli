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

import (
	"fmt"
	"io"
)

// Printer writes structured data to a writer in a specific format.
//
// Printer 将结构化数据以指定格式写入 writer。
type Printer interface {
	Print(w io.Writer, data interface{}) error
}

// Format represents an output format.
//
// Format 表示输出格式类型。
type Format string

const (
	FormatTable Format = "table"
	FormatWide  Format = "wide"
	FormatYAML  Format = "yaml"
	FormatJSON  Format = "json"
	FormatName  Format = "name"
)

// New returns a Printer for the given format string.
// "wide" is treated as an alias for "table"; the wider column set is
// determined by the caller passing a different row struct.
// "name" returns a NamePrinter with an empty ResourceType; callers that
// need a "type/name" prefix should set ResourceType after construction.
//
// 根据格式字符串返回对应的 Printer。
// "wide" 是 "table" 的别名，更宽的列集合由调用方传入不同的行结构体决定。
// "name" 返回 ResourceType 为空的 NamePrinter；需要 "type/name" 前缀的调用方
// 应在构造后自行设置 ResourceType。
func New(format string) (Printer, error) {
	switch Format(format) {
	case FormatTable, FormatWide, "":
		return &TablePrinter{}, nil
	case FormatYAML:
		return &YAMLPrinter{}, nil
	case FormatJSON:
		return &JSONPrinter{}, nil
	case FormatName:
		return &NamePrinter{}, nil
	default:
		return nil, fmt.Errorf("unknown output format %q (supported: table, wide, yaml, json, name)", format)
	}
}
