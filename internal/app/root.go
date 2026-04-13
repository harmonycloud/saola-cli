package app

import (
	// New kubectl-style verb commands.
	//
	// 新的 kubectl 风格动词命令。
	buildcmd "gitee.com/opensaola/saola-cli/internal/cmd/build"
	createcmd "gitee.com/opensaola/saola-cli/internal/cmd/create"
	deletecmd "gitee.com/opensaola/saola-cli/internal/cmd/delete"
	describecmd "gitee.com/opensaola/saola-cli/internal/cmd/describe"
	getcmd "gitee.com/opensaola/saola-cli/internal/cmd/get"
	inspectcmd "gitee.com/opensaola/saola-cli/internal/cmd/inspect"
	installcmd "gitee.com/opensaola/saola-cli/internal/cmd/install"
	runcmd "gitee.com/opensaola/saola-cli/internal/cmd/run"
	uninstallcmd "gitee.com/opensaola/saola-cli/internal/cmd/uninstall"
	upgradecmd "gitee.com/opensaola/saola-cli/internal/cmd/upgrade"
	versioncmd "gitee.com/opensaola/saola-cli/internal/cmd/version"

"gitee.com/opensaola/saola-cli/internal/config"
	"gitee.com/opensaola/saola-cli/internal/lang"
	"github.com/spf13/cobra"
)

// NewRootCmd builds the root cobra.Command for saola.
//
// 构建 saola 的根 cobra.Command。
func NewRootCmd() *cobra.Command {
	cfg := config.New()

	root := &cobra.Command{
		Use:   "saola",
		Short: lang.T("saola-cli — zeus-operator 中间件包管理命令行工具", "saola-cli — CLI companion for zeus-operator middleware management"),
		Long: lang.T(
			`saola 是 zeus-operator 的命令行伴侣工具，支持中间件包的打包与发布、Middleware 和 MiddlewareOperator 自定义资源管理、触发 Action 以及查询 Baseline。`,
			`saola is the CLI companion for zeus-operator. It allows you to pack and publish middleware packages, manage Middleware and MiddlewareOperator custom resources, trigger actions, and query baselines.`,
		),
		SilenceUsage:  true,
		SilenceErrors: true,
		// Resolve flag values into cfg before any sub-command runs.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return cfg.BindFlags(cmd.Root())
		},
	}

	// --- Persistent flags (available to all sub-commands) ---
	pf := root.PersistentFlags()
	pf.String("lang", "zh", lang.T("显示语言：zh（中文）| en（英文）", "Display language: zh (Chinese) | en (English)"))
	pf.String("kubeconfig", "", lang.T("kubeconfig 文件路径，默认 $KUBECONFIG 或 ~/.kube/config", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)"))
	pf.String("context", "", lang.T("使用的 kubeconfig context", "Kubeconfig context to use"))
	pf.StringP("namespace", "n", "", lang.T("中间件资源所在的 Kubernetes 命名空间", "Kubernetes namespace for middleware resources"))
	pf.String("pkg-namespace", config.DefaultPkgNamespace, lang.T("存放包 Secret 的命名空间", "Namespace where package Secrets are stored"))
	pf.String("log-level", "info", lang.T("日志详细级别：debug|info|warn|error", "Log verbosity: debug|info|warn|error"))
	pf.Bool("no-color", false, lang.T("禁用彩色输出", "Disable colored output"))

	// --- New kubectl-style verb commands ---
	//
	// 新的 kubectl 风格动词命令。
	root.AddCommand(
		getcmd.NewCmdGet(cfg),
		createcmd.NewCmdCreate(cfg),
		deletecmd.NewCmdDelete(cfg),
		describecmd.NewCmdDescribe(cfg),
		runcmd.NewCmdRun(cfg),
		installcmd.NewCmdInstall(cfg),
		uninstallcmd.NewCmdUninstall(cfg),
		upgradecmd.NewCmdUpgrade(cfg),
		buildcmd.NewCmdBuild(cfg),
		inspectcmd.NewCmdInspect(cfg),
		versioncmd.NewCmdVersion(cfg),
	)


	// Patch cobra's auto-generated English-only text to be bilingual.
	//
	// 将 cobra 自动生成的纯英文文字替换为中英双语。
	patchCobraBuiltins(root)

	return root
}

// patchCobraBuiltins bilingualize cobra's built-in text that cannot be set
// via the standard Short/Long/flag-usage fields:
//   - "completion" and "help" sub-command Short descriptions
//   - "-h, --help" flag usage on every command
//   - The "Use X --help" footer in the usage template
//
// patchCobraBuiltins 将 cobra 内置的纯英文文本双语化：
//   - "completion" 和 "help" 子命令的 Short 说明
//   - 每个命令的 "-h, --help" flag 用法说明
//   - 用法模板底部的 "Use X --help" 提示行
func patchCobraBuiltins(root *cobra.Command) {
	// 1. Bilingualize the usage-template footer.
	//
	// 1. 双语化用法模板底部提示行。
	footer := lang.T(
		`使用 "{{.CommandPath}} [command] --help" 查看子命令的详细说明`,
		`Use "{{.CommandPath}} [command] --help" for more information about a command.`,
	)
	root.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

` + footer + `
{{end}}
`)

	// 2. Patch "completion" and "help" built-in command descriptions.
	//    cobra adds these lazily; force-init them before patching.
	//
	// 2. 修正 "completion" 和 "help" 内置子命令描述。
	//    cobra 延迟添加这两个命令，需先强制初始化再修改。
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	for _, c := range root.Commands() {
		switch c.Name() {
		case "completion":
			c.Short = lang.T("生成 shell 自动补全脚本", "Generate the autocompletion script for the specified shell")
			c.Long = lang.T(
				`为指定 shell 生成自动补全脚本并输出到标准输出。安装后可在终端中使用 Tab 键补全 saola 命令和参数。`,
				`Generate the autocompletion script for the specified shell and print it to stdout.
After installing the script, Tab-completion for saola commands and flags will be available.`,
			)
		case "help":
			c.Short = lang.T("显示命令帮助信息", "Help about any command")
		}
	}

	// 3. Recursively patch "-h, --help" flag usage on every command.
	//    cobra adds the help flag lazily; InitDefaultHelpFlag forces it.
	//
	// 3. 递归修正每个命令的 "-h, --help" flag 说明。
	//    cobra 延迟添加 help flag，InitDefaultHelpFlag 强制初始化。
	var patchHelp func(*cobra.Command)
	patchHelp = func(c *cobra.Command) {
		c.InitDefaultHelpFlag()
		if f := c.Flags().Lookup("help"); f != nil {
			f.Usage = lang.T("显示帮助信息", "Show help for this command")
		}
		for _, sub := range c.Commands() {
			patchHelp(sub)
		}
	}
	patchHelp(root)
}

// Execute is the single entry point called from main.
//
// Execute 是从 main 调用的单一入口点。
func Execute() error {
	return NewRootCmd().Execute()
}
