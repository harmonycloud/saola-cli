package lang

import (
	"os"
	"strings"
)

// current holds the selected display language: "zh" (default) or "en".
//
// current 保存当前选择的显示语言："zh"（默认）或 "en"。
var current = "zh"

func init() {
	// Pre-scan os.Args for --lang before cobra parses flags.
	// cobra processes --help before PersistentPreRunE runs,
	// so we must detect language early via raw argument scanning.
	//
	// 在 cobra 解析 flag 之前预扫描 os.Args。
	// cobra 在 PersistentPreRunE 之前就处理 --help，
	// 因此必须通过原始参数扫描提前检测语言设置。
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--lang" && i+1 < len(args) {
			current = args[i+1]
			break
		}
		if strings.HasPrefix(arg, "--lang=") {
			current = strings.TrimPrefix(arg, "--lang=")
			break
		}
	}
}

// T returns zh when the display language is Chinese (default),
// or en when --lang=en is specified.
//
// T 在显示语言为中文（默认）时返回 zh，
// 指定 --lang=en 时返回 en。
func T(zh, en string) string {
	if current == "en" {
		return en
	}
	return zh
}
