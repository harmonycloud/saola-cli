# Saola CLI

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)

[English](README.md) | **中文**

[OpenSaola](https://gitee.com/opensaola/opensaola) 的命令行管理工具 — 管理 Kubernetes 上的中间件生命周期：包安装、实例创建、状态查询、版本升级与资源清理。

## 特性

- **kubectl 风格** — `get`/`create`/`delete`/`describe` 等动词 + 资源类型，学习成本低
- **交互式创建** — TUI 表单引导选择 Baseline、填写参数，无需手写 YAML
- **中英双语** — `--lang zh|en` 切换所有命令帮助和输出信息
- **包管理** — 打包、安装、升级、卸载中间件包（zstd 压缩 TAR 格式）
- **实例升级** — 通过 annotation 触发 controller 执行滚动升级，支持 `--wait` 等待完成
- **多格式输出** — `table`/`yaml`/`json`/`wide`/`name`，便于脚本集成

## 安装

### 从源码构建

```bash
git clone https://gitee.com/opensaola/saola-cli.git
cd saola-cli
make build
```

构建产物在 `bin/saola`，拷贝到 `$PATH` 即可使用：

```bash
cp bin/saola /usr/local/bin/
```

### go install

```bash
go install gitee.com/opensaola/saola-cli/cmd/saola@latest
```

## 快速上手

```bash
# 1. 安装中间件包
saola install ./redis-pkg -n middleware-operator --wait 5m

# 2. 交互式创建 Middleware 和 MiddlewareOperator
saola create

# 3. 查看实例状态
saola get middleware -n my-ns
saola get operator -n my-ns

# 4. 查看实例详情
saola describe middleware my-redis -n my-ns

# 5. 升级实例版本
saola upgrade middleware my-redis --to-version 2.0.0 -n my-ns --wait 5m

# 6. 删除实例
saola delete middleware my-redis -n my-ns
```

## 命令参考

### 顶级命令

| 命令 | 说明 |
|------|------|
| `create` | 从 YAML 文件或交互式 TUI 创建 Middleware / MiddlewareOperator |
| `get` | 列出或查看资源 |
| `describe` | 显示资源详细信息（Spec、Status、Conditions） |
| `delete` | 删除资源 |
| `run` | 触发 MiddlewareAction（一次性运维操作） |
| `upgrade` | 升级包或 Middleware / MiddlewareOperator 实例 |
| `install` | 安装中间件包到集群 |
| `uninstall` | 卸载中间件包 |
| `build` | 打包本地目录为 zstd 压缩 TAR（不安装） |
| `inspect` | 查看已安装包的内容和元数据 |
| `version` | 显示版本信息 |

### 资源子命令

以 `get` 为例，其他动词（`describe`/`delete`/`upgrade`）结构类似：

| 子命令 | 别名 | 说明 |
|--------|------|------|
| `get middleware [name]` | `mw` | 列出或查看 Middleware |
| `get operator [name]` | `op` | 列出或查看 MiddlewareOperator |
| `get action [name]` | `act` | 列出或查看 MiddlewareAction |
| `get baseline [name]` | `bl` | 查看已安装包中的 Baseline |
| `get package [name]` | `pkg` | 列出或查看已安装包 |
| `get all` | - | 聚合输出 middleware + operator + action |

### upgrade 子命令

```bash
# 包升级（替换已安装的 Package Secret）
saola upgrade <pkg-dir>

# 实例升级（通过 annotation 触发 controller）
saola upgrade middleware <name> --to-version <version> [--baseline <bl>] [--wait 5m]
saola upgrade operator <name>   --to-version <version> [--baseline <bl>] [--wait 5m]
```

### run 命令

```bash
# 触发 Action（如备份、恢复）
saola run <action-name> --middleware <mw-name> --params key1=val1,key2=val2 -n my-ns
```

## 全局 Flags

| Flag | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--kubeconfig` | - | `$KUBECONFIG` 或 `~/.kube/config` | kubeconfig 文件路径 |
| `--context` | - | - | kubeconfig context |
| `--namespace` | `-n` | - | 目标命名空间 |
| `--pkg-namespace` | - | `middleware-operator` | 包 Secret 所在命名空间 |
| `--lang` | - | `zh` | 语言：`zh`（中文）/ `en`（英文） |
| `--no-color` | - | `false` | 禁用彩色输出 |

常用资源命令还支持：

| Flag | 简写 | 说明 |
|------|------|------|
| `--all-namespaces` | `-A` | 列出所有命名空间的资源 |
| `--output` | `-o` | 输出格式：`table`/`yaml`/`json`/`wide`/`name` |
| `--wait` | - | 等待操作完成的超时时间（如 `5m`） |
| `--dry-run` | - | 预览变更，不实际执行 |

## 配置

### 环境变量

| 变量 | 说明 |
|------|------|
| `KUBECONFIG` | kubeconfig 文件路径 |
| `SAOLA_NAMESPACE` | 默认命名空间 |
| `SAOLA_PKG_NAMESPACE` | 包 Secret 命名空间（默认 `middleware-operator`） |

### 优先级

CLI Flag > 环境变量 > 默认值

## 项目结构

```
saola-cli/
├── cmd/saola/          # 入口
├── internal/
│   ├── app/            # 根命令注册
│   ├── cmd/            # 所有子命令实现
│   │   ├── action/     # MiddlewareAction
│   │   ├── baseline/   # Baseline 查询
│   │   ├── build/      # 包构建
│   │   ├── create/     # 资源创建（含交互式 TUI）
│   │   ├── delete/     # 资源删除
│   │   ├── describe/   # 资源详情
│   │   ├── get/        # 资源列表
│   │   ├── inspect/    # 包检查
│   │   ├── install/    # 包安装
│   │   ├── middleware/  # Middleware 资源组
│   │   ├── operator/   # MiddlewareOperator 资源组
│   │   ├── pkgcmd/     # 包管理命令
│   │   ├── upgrade/    # 升级
│   │   ├── uninstall/  # 卸载
│   │   ├── run/        # Action 执行
│   │   └── version/    # 版本信息
│   ├── client/         # Kubernetes client 封装
│   ├── config/         # 配置管理
│   ├── consts/         # 项目常量
│   ├── lang/           # 中英双语支持
│   ├── packager/       # TAR/zstd 打包解包
│   ├── printer/        # 输出格式化（table/yaml/json）
│   ├── version/        # 版本信息注入
│   └── waiter/         # 异步等待逻辑
├── Makefile
└── go.mod
```

## 构建与测试

```bash
# 构建
make build          # 编译到 bin/saola

# 测试
make test           # 运行单元测试
make lint           # go vet 静态检查

# 其他
make tidy           # 整理 go modules
make clean          # 清理构建产物
```

版本信息通过 ldflags 注入：

```bash
saola version
# Version: v0.1.0
# Git Commit: 6bd4f94
# Build Date: 2026-03-31T07:46:11Z
```

## 管理的 CRD 类型

Saola 通过 [OpenSaola](https://gitee.com/opensaola/opensaola) Operator 管理以下 Kubernetes 自定义资源：

| CRD | 缩写 | 说明 |
|-----|------|------|
| Middleware | `mid` | 中间件实例 |
| MiddlewareOperator | `mo` | 管理某类中间件的 Operator 实例 |
| MiddlewareAction | `ma` | 一次性运维操作（备份、恢复等） |
| MiddlewareBaseline | `mb` | Middleware 基线配置模板 |
| MiddlewareOperatorBaseline | `mob` | Operator 基线配置模板 |
| MiddlewarePackage | `mp` | 中间件分发包 |

## 文档

- [使用文档](docs/saola-cli-usage.md) | [Usage Guide (English)](docs/saola-cli-usage_en.md)
- [OpenSaola 技术文档](https://gitee.com/opensaola/opensaola/blob/master/docs/opensaola-technical.md)
- [包适配指南](https://gitee.com/opensaola/opensaola/blob/master/docs/opensaola-packaging.md)
- [故障排查指南](https://gitee.com/opensaola/opensaola/blob/master/docs/troubleshooting_zh.md)

## 参与贡献

欢迎贡献！请参阅 [CONTRIBUTING.md](CONTRIBUTING.md) 了解贡献指南。

## 许可证

Copyright 2025 The OpenSaola Authors.

基于 Apache License 2.0 许可证开源。详见 [LICENSE](LICENSE)。
