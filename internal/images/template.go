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

	"github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"
)

type templateValues struct {
	Values     map[string]any
	Globe      map[string]any
	Necessary  map[string]any
	Parameters map[string]any
	Step       map[string]any
}

func renderImageTemplate(text string, values templateValues) (string, error) {
	funcs := templateFuncs()
	var tpl *template.Template
	funcs["include"] = func(name string, data any) (string, error) {
		if tpl == nil {
			return "", fmt.Errorf("include template %q before parsing completed", name)
		}
		var included bytes.Buffer
		if err := tpl.ExecuteTemplate(&included, name, data); err != nil {
			return "", err
		}
		return included.String(), nil
	}

	var err error
	tpl, err = template.New("image").Option("missingkey=default").Funcs(funcs).Parse(text)
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
	funcs := sprig.HermeticTxtFuncMap()
	funcs["dict"] = dictFunc
	funcs["set"] = setFunc
	funcs["default"] = defaultFunc
	funcs["contains"] = strings.Contains
	funcs["replace"] = replaceFunc
	funcs["hasKey"] = hasKeyFunc
	funcs["toYaml"] = toYAMLFunc
	funcs["fromYaml"] = fromYAMLFunc
	funcs["fromYamlArray"] = fromYAMLArrayFunc
	funcs["toJson"] = toJSONFunc
	funcs["fromJson"] = fromJSONFunc
	funcs["fromJsonArray"] = fromJSONArrayFunc
	funcs["lookup"] = func(string, string, string, string) map[string]any {
		return map[string]any{}
	}
	funcs["required"] = func(_ string, val any) any {
		return val
	}
	funcs["fail"] = func(string) string {
		return ""
	}
	funcs["tpl"] = func(tplText string, vals any) (string, error) {
		switch typed := vals.(type) {
		case templateValues:
			return renderImageTemplate(tplText, typed)
		case *templateValues:
			return renderImageTemplate(tplText, *typed)
		default:
			return tplText, nil
		}
	}
	return funcs
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

func toYAMLFunc(v any) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

func fromYAMLFunc(str string) map[string]any {
	out := map[string]any{}
	if err := yaml.Unmarshal([]byte(str), &out); err != nil {
		out["Error"] = err.Error()
	}
	return out
}

func fromYAMLArrayFunc(str string) []any {
	var out []any
	if err := yaml.Unmarshal([]byte(str), &out); err != nil {
		return []any{err.Error()}
	}
	return out
}

func toJSONFunc(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func fromJSONFunc(str string) map[string]any {
	out := map[string]any{}
	if err := json.Unmarshal([]byte(str), &out); err != nil {
		out["Error"] = err.Error()
	}
	return out
}

func fromJSONArrayFunc(str string) []any {
	var out []any
	if err := json.Unmarshal([]byte(str), &out); err != nil {
		return []any{err.Error()}
	}
	return out
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
