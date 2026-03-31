package printer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// New("table") should return a *TablePrinter without error.
//
// New("table") 应返回 *TablePrinter 且不报错。
func TestNew_Table(t *testing.T) {
	p, err := New("table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*TablePrinter); !ok {
		t.Fatalf("expected *TablePrinter, got %T", p)
	}
}

// New("yaml") should return a *YAMLPrinter without error.
//
// New("yaml") 应返回 *YAMLPrinter 且不报错。
func TestNew_YAML(t *testing.T) {
	p, err := New("yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*YAMLPrinter); !ok {
		t.Fatalf("expected *YAMLPrinter, got %T", p)
	}
}

// New("json") should return a *JSONPrinter without error.
//
// New("json") 应返回 *JSONPrinter 且不报错。
func TestNew_JSON(t *testing.T) {
	p, err := New("json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*JSONPrinter); !ok {
		t.Fatalf("expected *JSONPrinter, got %T", p)
	}
}

// New("invalid") should return a non-nil error.
//
// New("invalid") 应返回非空错误。
func TestNew_Invalid(t *testing.T) {
	_, err := New("invalid")
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}

// TablePrinter.Print with [][]string should output the header row and data row.
//
// TablePrinter.Print 传入 [][]string 时应输出表头行和数据行。
func TestTablePrinter_PrintStringSlice(t *testing.T) {
	p := &TablePrinter{}
	rows := [][]string{
		{"NAME", "AGE"},
		{"alice", "30"},
	}
	var buf bytes.Buffer
	if err := p.Print(&buf, rows); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Errorf("output missing header NAME: %q", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("output missing data row 'alice': %q", out)
	}
}

// TablePrinter.Print with a struct slice should use exported field names as headers.
//
// TablePrinter.Print 传入 struct 切片时应以导出字段名作为列头。
func TestTablePrinter_PrintStructSlice(t *testing.T) {
	type Item struct {
		Name  string
		Value int
	}
	p := &TablePrinter{}
	data := []Item{
		{Name: "foo", Value: 42},
		{Name: "bar", Value: 99},
	}
	var buf bytes.Buffer
	if err := p.Print(&buf, data); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}
	out := buf.String()
	// Headers should be uppercased field names.
	// 列头应为大写的字段名。
	if !strings.Contains(out, "NAME") {
		t.Errorf("output missing header NAME: %q", out)
	}
	if !strings.Contains(out, "VALUE") {
		t.Errorf("output missing header VALUE: %q", out)
	}
	if !strings.Contains(out, "foo") {
		t.Errorf("output missing row data 'foo': %q", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("output missing row data '42': %q", out)
	}
}

// YAMLPrinter.Print should produce valid YAML that round-trips back to the original struct.
//
// YAMLPrinter.Print 应输出合法的 YAML，可以反序列化回原始结构体。
func TestYAMLPrinter_Print(t *testing.T) {
	type Payload struct {
		Key   string `yaml:"key"`
		Count int    `yaml:"count"`
	}
	p := &YAMLPrinter{}
	input := Payload{Key: "hello", Count: 7}
	var buf bytes.Buffer
	if err := p.Print(&buf, input); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}
	var got Payload
	if err := yaml.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid YAML: %v\nraw: %s", err, buf.String())
	}
	if got.Key != input.Key || got.Count != input.Count {
		t.Errorf("round-trip mismatch: want %+v, got %+v", input, got)
	}
}

// JSONPrinter.Print should produce valid, indented JSON that round-trips back to the original struct.
//
// JSONPrinter.Print 应输出合法的缩进 JSON，可以反序列化回原始结构体。
func TestJSONPrinter_Print(t *testing.T) {
	type Payload struct {
		Key   string `json:"key"`
		Count int    `json:"count"`
	}
	p := &JSONPrinter{}
	input := Payload{Key: "world", Count: 3}
	var buf bytes.Buffer
	if err := p.Print(&buf, input); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}
	var got Payload
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	if got.Key != input.Key || got.Count != input.Count {
		t.Errorf("round-trip mismatch: want %+v, got %+v", input, got)
	}
	// Verify indentation is present (indented output contains newlines).
	// 验证输出包含换行（缩进格式必然包含换行符）。
	if !strings.Contains(buf.String(), "\n") {
		t.Errorf("expected indented JSON with newlines, got: %s", buf.String())
	}
}
