package printer

import (
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLPrinter renders data as YAML.
//
// YAMLPrinter 将数据渲染为 YAML 格式。
type YAMLPrinter struct{}

func (p *YAMLPrinter) Print(w io.Writer, data interface{}) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(data)
}
