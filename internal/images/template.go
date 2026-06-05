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

package images

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

type templateValues struct {
	Values     map[string]any
	Globe      map[string]any
	Necessary  map[string]any
	Parameters map[string]any
	Step       map[string]any
}

func renderImageTemplate(text string, values templateValues) (string, error) {
	tpl, err := template.New("image").Funcs(templateFuncs()).Parse(text)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err = tpl.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"dict":     dictFunc,
		"set":      setFunc,
		"default":  defaultFunc,
		"contains": strings.Contains,
		"replace":  replaceFunc,
		"hasKey":   hasKeyFunc,
		"toJson":   toJSONFunc,
	}
}

func dictFunc(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict expects an even number of arguments")
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			key = fmt.Sprint(values[i])
		}
		m[key] = values[i+1]
	}
	return m, nil
}

func setFunc(m map[string]any, key string, value any) string {
	m[key] = value
	return ""
}

func defaultFunc(def, value any) any {
	if isEmpty(value) {
		return def
	}
	return value
}

func hasKeyFunc(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

func replaceFunc(old, new, src string) string {
	return strings.ReplaceAll(src, old, new)
}

func toJSONFunc(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch typed := v.(type) {
	case string:
		return typed == ""
	case bool:
		return !typed
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return rv.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Invalid:
		return true
	default:
		return false
	}
}
