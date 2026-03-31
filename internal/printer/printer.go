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
	FormatYAML  Format = "yaml"
	FormatJSON  Format = "json"
)

// New returns a Printer for the given format string.
// Defaults to table if format is empty or unrecognised.
//
// 根据格式字符串返回对应的 Printer；格式为空或无法识别时默认返回 table 格式。
func New(format string) (Printer, error) {
	switch Format(format) {
	case FormatTable, "":
		return &TablePrinter{}, nil
	case FormatYAML:
		return &YAMLPrinter{}, nil
	case FormatJSON:
		return &JSONPrinter{}, nil
	default:
		return nil, fmt.Errorf("unknown output format %q (supported: table, yaml, json)", format)
	}
}
