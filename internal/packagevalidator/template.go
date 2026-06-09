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

package packagevalidator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"
)

const recursionLimit = 1000

type templateValues struct {
	Values       map[string]any
	Globe        map[string]any
	Necessary    map[string]any
	Step         map[string]any
	Capabilities capabilities
	Parameters   map[string]any
}

type capabilities struct {
	KubeVersion kubeVersion
	APIVersions apiVersions
}

type kubeVersion struct {
	Version    string
	Major      string
	Minor      string
	GitVersion string
}

type apiVersions struct{}

func (apiVersions) Has(string) bool {
	return true
}

func defaultTemplateValues() templateValues {
	return templateValues{
		Values: map[string]any{
			"agentsMd":   "package validation\n",
			"name":       "saola-validate",
			"skillsList": []string{"package-validation"},
		},
		Globe: map[string]any{
			"Name":           "saola-validate",
			"Namespace":      "default",
			"Labels":         map[string]string{"app": "saola-validate"},
			"Annotations":    map[string]string{"middleware.cn/validate": "true"},
			"PackageName":    "saola-validate-1.0.0",
			"MiddlewareName": "saola-validate",
			"repository":     "registry.example.com",
			"project":        "opensaola",
		},
		Necessary: map[string]any{
			"image":      "registry.example.com/opensaola/validate:1.0.0",
			"repository": "registry.example.com",
			"version":    "1.0.0",
			"resource": map[string]any{
				"requests": map[string]any{"cpu": "100m", "memory": "128Mi"},
				"limits":   map[string]any{"cpu": "1", "memory": "512Mi"},
			},
		},
		Step:       map[string]any{},
		Parameters: map[string]any{},
		Capabilities: capabilities{
			KubeVersion: kubeVersion{
				Version:    "v1.30.0",
				Major:      "1",
				Minor:      "30",
				GitVersion: "v1.30.0",
			},
			APIVersions: apiVersions{},
		},
	}
}

func renderTemplate(text string, values templateValues) (string, error) {
	tpl := template.New("package-validate").Option("missingkey=default")
	includedNames := map[string]int{}
	funcs := validationFuncMap()
	funcs["include"] = includeFunc(tpl, includedNames)
	funcs["tpl"] = tplFunc(tpl, includedNames)
	parsed, err := tpl.Funcs(funcs).Parse(text)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := parsed.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func validationFuncMap() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	delete(funcs, "env")
	delete(funcs, "expandenv")
	funcs["toYaml"] = toYAML
	funcs["fromYaml"] = fromYAML
	funcs["fromYamlArray"] = fromYAMLArray
	funcs["toJson"] = toJSON
	funcs["fromJson"] = fromJSON
	funcs["fromJsonArray"] = fromJSONArray
	funcs["toCue"] = toCue
	funcs["include"] = func(string, any) string { return "" }
	funcs["tpl"] = func(string, any) string { return "" }
	funcs["lookup"] = func(string, string, string, string) (map[string]any, error) {
		return map[string]any{}, nil
	}
	funcs["required"] = func(warn string, val any) (any, error) {
		if isEmpty(val) {
			return val, errors.New(warn)
		}
		return val, nil
	}
	funcs["fail"] = func(msg string) (string, error) {
		return "", errors.New(msg)
	}
	return funcs
}

func toYAML(v any) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

func fromYAML(str string) map[string]any {
	out := map[string]any{}
	if err := yaml.Unmarshal([]byte(str), &out); err != nil {
		out["Error"] = err.Error()
	}
	return out
}

func fromYAMLArray(str string) []any {
	var out []any
	if err := yaml.Unmarshal([]byte(str), &out); err != nil {
		return []any{err.Error()}
	}
	return out
}

func toJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func fromJSON(str string) map[string]any {
	out := map[string]any{}
	if err := json.Unmarshal([]byte(str), &out); err != nil {
		out["Error"] = err.Error()
	}
	return out
}

func fromJSONArray(str string) []any {
	var out []any
	if err := json.Unmarshal([]byte(str), &out); err != nil {
		return []any{err.Error()}
	}
	return out
}

func includeFunc(t *template.Template, includedNames map[string]int) func(string, any) (string, error) {
	return func(name string, data any) (string, error) {
		var buf strings.Builder
		if count := includedNames[name]; count > recursionLimit {
			return "", fmt.Errorf("rendering template has a nested reference name: %s", name)
		}
		includedNames[name]++
		err := t.ExecuteTemplate(&buf, name, data)
		includedNames[name]--
		return buf.String(), err
	}
}

func tplFunc(parent *template.Template, includedNames map[string]int) func(string, any) (string, error) {
	return func(tplText string, vals any) (string, error) {
		t, err := parent.Clone()
		if err != nil {
			return "", fmt.Errorf("cannot clone template: %w", err)
		}
		t.Funcs(template.FuncMap{
			"include": includeFunc(t, includedNames),
			"tpl":     tplFunc(t, includedNames),
		})
		t, err = t.New(parent.Name()).Parse(tplText)
		if err != nil {
			return "", fmt.Errorf("cannot parse template %q: %w", tplText, err)
		}
		var buf strings.Builder
		if err := t.Execute(&buf, vals); err != nil {
			return "", fmt.Errorf("error during tpl function execution for %q: %w", tplText, err)
		}
		return strings.ReplaceAll(buf.String(), "<no value>", ""), nil
	}
}

func toCue(data any) string {
	var buf strings.Builder
	var write func(any)
	write = func(v any) {
		if v == nil {
			buf.WriteString("null")
			return
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			buf.WriteString("{")
			keys := rv.MapKeys()
			for i, key := range keys {
				if i > 0 {
					buf.WriteString(", ")
				}
				fmt.Fprintf(&buf, "%q: ", fmt.Sprint(key.Interface()))
				write(rv.MapIndex(key).Interface())
			}
			buf.WriteString("}")
		case reflect.Slice, reflect.Array:
			buf.WriteString("[")
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					buf.WriteString(", ")
				}
				write(rv.Index(i).Interface())
			}
			buf.WriteString("]")
		case reflect.String:
			fmt.Fprintf(&buf, "%q", rv.String())
		case reflect.Bool:
			fmt.Fprintf(&buf, "%t", rv.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fmt.Fprintf(&buf, "%d", rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fmt.Fprintf(&buf, "%d", rv.Uint())
		case reflect.Float32, reflect.Float64:
			fmt.Fprintf(&buf, "%g", rv.Float())
		default:
			fmt.Fprintf(&buf, "%v", v)
		}
	}
	write(data)
	return buf.String()
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
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
