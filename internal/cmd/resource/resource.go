// Package resource provides the resource type registry and alias resolution for saola CLI.
//
// resource 包提供 saola CLI 的资源类型注册表与别名解析功能。
package resource

// Resource type canonical names.
//
// 资源类型规范名称常量。
const (
	Middleware = "middleware"
	Operator   = "operator"
	Action     = "action"
	Baseline   = "baseline"
	Package    = "package"
	All        = "all"
)

// aliases maps short aliases to their canonical resource type names.
//
// aliases 将简写别名映射到对应的规范资源类型名称。
var aliases = map[string]string{
	"mw":  Middleware,
	"op":  Operator,
	"act": Action,
	"bl":  Baseline,
	"pkg": Package,
}

// Resolve returns the canonical resource type name for the given input.
// If input matches a known alias, the canonical name is returned.
// Otherwise the input is returned unchanged.
//
// Resolve 将用户输入（含别名）解析为规范资源类型名称。
// 若 input 匹配已知别名则返回规范名，否则原样返回。
func Resolve(input string) string {
	if canonical, ok := aliases[input]; ok {
		return canonical
	}
	return input
}
