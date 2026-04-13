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
	"reflect"
	"strings"
	"text/tabwriter"
)

// TablePrinter renders data as a tab-aligned table.
// It accepts:
//   - []map[string]string  — rows with arbitrary string columns
//   - [][]string           — header row + data rows (first element is header)
//   - any slice of structs — columns are exported fields, values are fmt.Sprint'd
//
// TablePrinter 将数据渲染为对齐的表格，支持：
//   - []map[string]string  — 任意字符串列的行
//   - [][]string           — 首行为表头 + 数据行
//   - 任意 struct 切片     — 列为导出字段，值通过 fmt.Sprint 格式化
type TablePrinter struct{}

func (p *TablePrinter) Print(w io.Writer, data interface{}) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	defer tw.Flush()

	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case [][]string:
		return printStringTable(tw, v)
	case []map[string]string:
		return printMapTable(tw, v)
	default:
		return printStructTable(tw, data)
	}
}

func printStringTable(w io.Writer, rows [][]string) error {
	for _, row := range rows {
		_, err := fmt.Fprintln(w, strings.Join(row, "\t"))
		if err != nil {
			return err
		}
	}
	return nil
}

func printMapTable(w io.Writer, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	// Collect ordered keys from the first row.
	keys := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		keys = append(keys, k)
	}
	// Print header.
	_, err := fmt.Fprintln(w, strings.Join(keys, "\t"))
	if err != nil {
		return err
	}
	for _, row := range rows {
		vals := make([]string, 0, len(keys))
		for _, k := range keys {
			vals = append(vals, row[k])
		}
		_, err = fmt.Fprintln(w, strings.Join(vals, "\t"))
		if err != nil {
			return err
		}
	}
	return nil
}

func printStructTable(w io.Writer, data interface{}) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		// Single item — print as key: value pairs.
		return printStructFields(w, val)
	}
	if val.Len() == 0 {
		return nil
	}
	elem := val.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	if elem.Kind() != reflect.Struct {
		for i := 0; i < val.Len(); i++ {
			_, err := fmt.Fprintln(w, fmt.Sprint(val.Index(i).Interface()))
			if err != nil {
				return err
			}
		}
		return nil
	}
	t := elem.Type()
	headers := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.IsExported() {
			headers = append(headers, strings.ToUpper(f.Name))
		}
	}
	_, err := fmt.Fprintln(w, strings.Join(headers, "\t"))
	if err != nil {
		return err
	}
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		vals := make([]string, 0, len(headers))
		for j := 0; j < item.NumField(); j++ {
			if item.Type().Field(j).IsExported() {
				vals = append(vals, fmt.Sprint(item.Field(j).Interface()))
			}
		}
		_, err = fmt.Fprintln(w, strings.Join(vals, "\t"))
		if err != nil {
			return err
		}
	}
	return nil
}

func printStructFields(w io.Writer, val reflect.Value) error {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		_, err := fmt.Fprintln(w, fmt.Sprint(val.Interface()))
		return err
	}
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.IsExported() {
			_, err := fmt.Fprintf(w, "%s\t%v\n", f.Name, val.Field(i).Interface())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
