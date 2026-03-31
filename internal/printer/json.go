package printer

import (
	"encoding/json"
	"io"
)

// JSONPrinter renders data as indented JSON.
//
// JSONPrinter 将数据渲染为缩进的 JSON 格式。
type JSONPrinter struct{}

func (p *JSONPrinter) Print(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
