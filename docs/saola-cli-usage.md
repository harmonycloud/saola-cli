# Saola CLI 使用文档

**中文** | [English](saola-cli-usage_en.md)

## 目录

- [1. 概览](#1-概览)
  - [1.1 saola-cli 是什么](#11-saola-cli-是什么)
  - [1.2 核心功能](#12-核心功能)
  - [1.3 与 OpenSaola / dataservice-baseline 的关系](#13-与-opensaola--dataservice-baseline-的关系)
  - [1.4 安装与构建](#14-安装与构建)
- [2. 全局配置](#2-全局配置)
  - [2.1 Kubeconfig 配置](#21-kubeconfig-配置)
  - [2.2 命名空间选项](#22-命名空间选项)
  - [2.3 输出格式（-o flag）](#23-输出格式-o-flag)
  - [2.4 其他全局 Flag](#24-其他全局-flag)
  - [2.5 环境变量](#25-环境变量)
- [3. 命令参考](#3-命令参考)
  - [3.1 包管理（saola pkg）](#31-包管理saola-pkg)
  - [3.2 中间件实例管理](#32-中间件实例管理)
  - [3.3 Operator 管理](#33-operator-管理)
  - [3.4 Baseline 管理](#34-baseline-管理)
  - [3.5 Action 管理](#35-action-管理)
  - [3.6 顶层命令](#36-顶层命令)
  - [3.7 中间件包镜像导出与导入](#37-中间件包镜像导出与导入)
  - [3.8 本地包校验](#38-本地包校验)
  - [3.9 命令帮助与 Shell 补全](#39-命令帮助与-shell-补全)
- [4. 包（Package）规范](#4-包package规范)
  - [4.1 TAR 包格式和目录结构](#41-tar-包格式和目录结构)
  - [4.2 metadata.yaml 要求](#42-metadatayaml-要求)
  - [4.3 包状态机](#43-包状态机)
- [5. Secret 结构](#5-secret-结构)
  - [5.1 包安装后生成的 Secret](#51-包安装后生成的-secret)
  - [5.2 Secret Labels 和 Annotations](#52-secret-labels-和-annotations)
  - [5.3 Secret data 字段](#53-secret-data-字段)
- [6. 完整工作流示例](#6-完整工作流示例)
  - [6.1 从零开始的完整流程](#61-从零开始的完整流程)
  - [6.2 包生命周期](#62-包生命周期)
  - [6.3 中间件实例生命周期](#63-中间件实例生命周期)
  - [6.4 Operator 生命周期](#64-operator-生命周期)
  - [6.5 中间件包与镜像离线交付](#65-中间件包与镜像离线交付)
- [7. 交互式创建详解](#7-交互式创建详解)
  - [7.1 交互流程](#71-交互流程)
  - [7.2 JSON Schema 支持](#72-json-schema-支持)
  - [7.3 参数验证](#73-参数验证)
- [8. 输出格式](#8-输出格式)
  - [8.1 Table 格式](#81-table-格式)
  - [8.2 JSON 格式](#82-json-格式)
  - [8.3 YAML 格式](#83-yaml-格式)
  - [8.4 Name 格式](#84-name-格式)
- [9. 配置与环境](#9-配置与环境)
  - [9.1 配置文件](#91-配置文件)
  - [9.2 环境变量](#92-环境变量)
- [10. 与其他项目的数据流](#10-与其他项目的数据流)
  - [10.1 saola-cli --> dataservice-baseline（消费包）](#101-saola-cli----dataservice-baseline消费包)
  - [10.2 saola-cli --> OpenSaola（创建 K8s 资源）](#102-saola-cli----opensaola创建-k8s-资源)
  - [10.3 完整数据流图](#103-完整数据流图)

---

## 1. 概览

### 1.1 saola-cli 是什么

saola-cli 是 OpenSaola 的命令行伴侣工具（CLI companion），用于管理 Kubernetes 集群中的中间件包（Package）、Middleware 和 MiddlewareOperator 自定义资源、触发 Action 运维操作，以及查询 Baseline 模板。

项目名称：`saola-cli`，编译产物二进制名称：`saola`。

### 1.2 核心功能

- **包管理（Package）**：将本地目录打包为 zstd 压缩 TAR 归档，安装/卸载/升级/列出/检查已安装的中间件包。
- **Middleware 实例管理**：创建、查询、描述、升级、删除 Middleware 自定义资源。
- **MiddlewareOperator 管理**：创建、查询、描述、升级、删除 MiddlewareOperator 自定义资源。
- **Baseline 查询**：列出和查看已安装包中嵌入的 MiddlewareBaseline / MiddlewareOperatorBaseline / MiddlewareActionBaseline。
- **Action 管理**：触发一次性运维操作（MiddlewareAction），查询和描述其执行状态。
- **交互式创建**：通过终端引导式表单（基于 charmbracelet/huh）选择 baseline、填写参数、预览 YAML 并确认创建。
- **多语言支持**：通过 `--lang` 切换中文（默认）/ 英文界面。
- **kubectl 风格快捷命令**：`saola get`、`saola create`、`saola delete`、`saola describe`、`saola run` 等。
- **包禁用与卸载分离**：`saola disable package <name>` 仅禁用包；`saola uninstall <name>` 会清理包资源并删除包 Secret。

### 1.3 与 OpenSaola / dataservice-baseline 的关系

| 项目 | 职责 | 与 saola-cli 的关系 |
|------|------|---------------------|
| **OpenSaola** | Kubernetes Operator，监听 CRD 变化并 reconcile 中间件生命周期 | saola-cli 创建的 Secret / CR 由 OpenSaola 消费和处理 |
| **dataservice-baseline** | 提供 40+ 中间件的 CUE/Go 模板、Baseline 定义、Action 模板 | saola-cli 打包的目录内容来源于 dataservice-baseline 构建产物 |

数据流向：
```
dataservice-baseline (模板/Baseline) --> 本地包目录 --> saola-cli pack --> K8s Secret --> OpenSaola reconcile
                                                        saola-cli create --> K8s Middleware/Operator CR --> OpenSaola reconcile
```

### 1.4 安装与构建

```bash
# 从源码构建
cd saola-cli
make build

# 二进制产物位于 bin/saola
./bin/saola version
```

构建时通过 `-ldflags` 注入版本信息：

| 变量 | 说明 |
|------|------|
| `Version` | 正式发布使用精确标签；本地 Makefile 构建使用 git describe，无法获取时回退到 `dev` |
| `GitCommit` | 完整 Git commit hash，无法获取时回退到 `unknown` |
| `BuildDate` | 来源提交时间转换为 UTC，无法获取时回退到 `unknown` |

---

## 2. 全局配置

### 2.1 Kubeconfig 配置

saola 使用标准的 kubeconfig 加载规则：

1. `--kubeconfig` flag 显式指定路径
2. `$KUBECONFIG` 环境变量
3. `~/.kube/config` 默认路径

可通过 `--context` 指定使用的 kubeconfig context。

### 2.2 命名空间选项

| Flag | 缩写 | 默认值 | 说明 |
|------|------|--------|------|
| `--namespace` | `-n` | 空（回退到 `$SAOLA_NAMESPACE`，再回退到 `default`） | 中间件资源所在的 Kubernetes 命名空间 |
| `--pkg-namespace` | 无 | `middleware-operator` | 存放包 Secret 的命名空间 |

> 注意：OpenSaola 源码中 DataNamespace 默认值为 `default`，实际部署环境通常通过配置覆盖为 `middleware-operator`。确保 saola-cli 的 --pkg-namespace 与 OpenSaola 的配置一致。

### 2.3 输出格式（-o flag）

大多数 `get` / `list` / `inspect` 命令支持 `-o` / `--output` 选项：

| 格式 | 说明 |
|------|------|
| `table` | 默认，对齐的表格输出 |
| `wide` | 扩展表格，额外显示 LABELS 列 |
| `yaml` | 完整 YAML 格式输出 |
| `json` | 缩进 JSON 格式输出 |
| `name` | 仅输出 `type/name`，类似 `kubectl get -o name` |

### 2.4 其他全局 Flag

| Flag | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--lang` | string | `zh` | 显示语言：`zh`（中文）或 `en`（英文） |
| `--log-level` | string | `info` | 日志级别：`debug`、`info`、`warn`、`error` |
| `--no-color` | bool | `false` | 禁用彩色输出 |
| `-h`, `--help` | bool | `false` | 显示帮助信息 |

### 2.5 环境变量

| 环境变量 | 对应 Flag | 说明 |
|----------|-----------|------|
| `KUBECONFIG` | `--kubeconfig` | kubeconfig 文件路径 |
| `SAOLA_NAMESPACE` | `--namespace` | 默认资源命名空间 |
| `SAOLA_PKG_NAMESPACE` | `--pkg-namespace` | 默认包 Secret 命名空间 |

> 注意：命令行 flag 优先于环境变量。环境变量仅在对应 flag 未显式设置时生效。

---

## 3. 命令参考

### 3.1 包管理（saola pkg）

包管理命令组通过 `saola package`（别名 `saola pkg`）访问，也可通过顶层快捷命令直接使用。

---

#### saola pkg build

**语法**: `saola package build <pkg-dir> [flags]`

**说明**: 将本地目录打包为 zstd 压缩的 TAR 文件，不执行安装。适用于 CI 流水线或离线分发场景。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<pkg-dir>` | - | 位置参数 | 必填 | 本地包目录路径 |
| `--output` | `-o` | string | `<name>-<version>.pkg` | 输出文件路径 |

**示例**:
```bash
# 打包当前目录
saola package build .

# 指定输出路径
saola package build ./my-redis --output ./dist/redis-v1.pkg
```

**注意事项**:
- 包目录必须包含 `metadata.yaml` 文件，且 `name` 和 `version` 为必填字段。
- 输出目录不存在时会自动创建。
- 隐藏文件和目录（以 `.` 开头）会被自动跳过。

---

#### saola pkg install

**语法**: `saola package install <pkg-dir> [flags]`

**说明**: 将本地包目录打包并在 `pkg-namespace` 中创建 Immutable Secret。OpenSaola 检测到 Secret 后会自动安装该包。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<pkg-dir>` | - | 位置参数 | 必填 | 本地包目录路径 |
| `--name` | - | string | `<name>-<version>` | 覆盖 Secret 名称 |
| `--wait` | - | duration | `0`（不等待） | 等待安装完成的超时时间（如 `5m`） |
| `--dry-run` | - | bool | `false` | 打包并打印目标 Secret 名称，不创建资源或输出完整 YAML |

**示例**:
```bash
# 从当前目录安装
saola package install .

# 指定名称并等待
saola package install ./my-redis --name redis-v1 --wait 5m

# 仅预览
saola package install . --dry-run
```

**注意事项**:
- 若同名 Secret 已存在，命令会报错并提示使用 `package upgrade`。
- `--wait` 通过轮询 Secret 的 `enabled` label 判断安装是否完成。
- 安装失败时，Secret 上的 `installError` annotation 会包含错误信息。

---

#### saola pkg disable

**语法**: `saola package disable <name>`

**说明**: 将包 Secret 标记为 `enabled=false`，不删除 Secret，也不清理已发布的 Baseline / Configuration / Action 等包资源。

**示例**:
```bash
saola package disable redis-v1
saola disable package redis-v1
```

**注意事项**:
- disable 是可逆的状态切换，不等价于卸载。
- 若包存在待处理的 install/uninstall 注解，命令会报错，需等待当前操作完成后再禁用。

---

#### saola pkg uninstall

**语法**: `saola package uninstall <name> [flags]`

**说明**: 请求真实卸载包。CLI 会先检查是否存在引用该包的 Middleware 或 MiddlewareOperator；若存在会报错并提示先删除对应实例。检查通过后，CLI 会给包 Secret 添加 `middleware.cn/package-secret-cleanup` finalizer 并发起删除；OpenSaola 在 Secret 处于 Terminating 状态时清理包发布的资源，最后移除 finalizer，让 Kubernetes 删除包 Secret。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | 包 Secret 名称 |
| `--wait` | - | duration | `0`（不等待） | 等待卸载完成的超时时间 |

**示例**:
```bash
saola package uninstall redis-v1
saola package uninstall redis-v1 --wait 5m
```

**注意事项**:
- 卸载前必须先删除已部署的 Middleware / MiddlewareOperator。
- `--wait` 的完成条件是包 Secret 被删除（NotFound）。
- 若 OpenSaola 二次检查发现包仍被使用，会在 Secret 上写入 `middleware.cn/uninstallError` 并保留 finalizer；删除引用后，Operator 后续重试会继续清理。

---

#### saola pkg upgrade

**语法**: `saola package upgrade <pkg-dir> [flags]`

**说明**: 用本地目录的新内容替换包 Secret。由于 Immutable Secret 无法原地更新，命令会先删除旧 Secret 再以新数据重新创建。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<pkg-dir>` | - | 位置参数 | 必填 | 本地包目录路径 |
| `--name` | - | string | `<name>-<version>` | 覆盖 Secret 名称 |
| `--wait` | - | duration | `0`（不等待） | 升级后等待安装完成的超时时间 |

**示例**:
```bash
saola package upgrade ./my-redis
saola package upgrade ./my-redis --name redis-custom
saola package upgrade ./my-redis --wait 5m
```

**注意事项**:
- 旧 Secret 不存在时，效果等同于全新安装。
- 升级过程中存在短暂的 Secret 空窗期（删除-重建之间）。

---

#### saola pkg inspect

**语法**: `saola package inspect <name> [flags]`

**说明**: 从 `pkg-namespace` 中读取指定包的 Secret，解压 TAR 归档并展示包内文件列表及元数据。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | 包 Secret 名称 |
| `--output` | `-o` | string | `table` | 输出格式：`table`、`yaml`、`json` |

**示例**:
```bash
saola package inspect redis-v1
saola package inspect redis-v1 -o yaml
```

**注意事项**:
- `table` 格式输出元数据字段和文件列表（含每个文件的大小）。
- `yaml` / `json` 格式输出完整的包数据结构。

---

#### saola pkg list

**语法**: `saola package list [flags]`（别名：`saola package ls`）

**说明**: 列出 `pkg-namespace` 中所有已安装的中间件包 Secret。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--component` | - | string | 空 | 按组件名过滤 |
| `--version` | - | string | 空 | 按包版本过滤 |
| `--output` | `-o` | string | `table` | 输出格式：`table`、`yaml`、`json` |

**示例**:
```bash
saola package list
saola package list --component redis
saola package list -o yaml
```

**注意事项**:
- 表格输出列：`NAME`、`COMPONENT`、`VERSION`、`ENABLED`、`CREATED`。
- 无包时输出 "No packages found."。

---

### 3.2 中间件实例管理

使用 `saola get middleware`、`saola describe middleware`、`saola upgrade middleware` 和 `saola delete middleware` 管理实例；资源类型可简写为 `mw`。创建使用 `saola create -f <file>`。

---

#### saola create — Middleware

**语法**: `saola create -f <file> [flags]`

**说明**: 从指定的 YAML 文件读取 Middleware 清单并在集群中创建。命令会自动从已安装包中查找匹配 `spec.baseline` 的 MiddlewareBaseline，并补全以下 labels 和 `spec.operatorBaseline`：

- `middleware.cn/packagename`
- `middleware.cn/packageversion`
- `middleware.cn/component`
- `middleware.cn/definition`

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--file` | `-f` | string | 必填 | YAML 清单文件路径 |
| `--namespace` | `-n` | string | 空（取 manifest 中的值） | 覆盖清单中的 namespace |

**示例**:
```bash
saola create -f middleware.yaml
saola create -f middleware.yaml -n production
```

**注意事项**:
- `spec.baseline` 为必填字段，用于定位匹配的包和 baseline。
- 若已安装包中未找到匹配的 MiddlewareBaseline，命令会报错。
- 创建后，若 `spec.operatorBaseline.name` 非空但对应的 MiddlewareOperator 不存在，会打印警告。
- Middleware 不会自动创建 MiddlewareOperator，需独立创建。

---

#### saola get middleware

**语法**: `saola get middleware [name] [flags]`

**说明**: 列出命名空间内的所有 Middleware 资源，或按名称获取单个资源。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `[name]` | - | 位置参数 | 可选 | 指定则获取单个资源 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |
| `--all-namespaces` | `-A` | bool | `false` | 跨所有命名空间列出 |
| `--output` | `-o` | string | `table` | 输出格式：`table`、`wide`、`yaml`、`json`、`name` |

**示例**:
```bash
saola get middleware
saola get middleware my-redis
saola get middleware my-redis -o yaml
saola get middleware -A -o json
```

**注意事项**:
- 表格输出列：`NAME`、`NAMESPACE`、`BASELINE`、`STATE`、`AGE`。
- `wide` 格式额外显示 `LABELS` 列。
- `name` 格式输出 `middleware/<name>`。

---

#### saola describe middleware

**语法**: `saola describe middleware <name> [flags]`

**说明**: 获取单个 Middleware 资源并以可读格式输出其 spec、status 和 conditions。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | Middleware 资源名称 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |

**示例**:
```bash
saola describe middleware my-redis
saola describe middleware my-redis -n production
```

**输出内容**:
- Metadata：Name、Namespace、Age、Labels、Annotations
- Spec：Baseline、OperatorBaseline（Name/GvkName）、Configurations、PreActions
- Status：State、Reason、ObservedGeneration、CustomResources（Type/Phase/Replicas/Reason）
- Conditions：TYPE、STATUS、REASON、MESSAGE

---

#### saola upgrade middleware

**语法**: `saola upgrade middleware <name> [flags]`

**说明**: 通过在 Middleware CR 上设置 `middleware.cn/update` 和 `middleware.cn/baseline` 注解触发 OpenSaola 执行版本升级。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | Middleware 实例名称 |
| `--to-version` | - | string | 必填 | 目标版本号 |
| `--baseline` | - | string | 空（保持当前值） | 目标 baseline 名称 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |
| `--wait` | - | duration | `0`（不等待） | 等待升级完成的超时时间 |

**示例**:
```bash
saola upgrade middleware my-redis --to-version 7.2.1
saola upgrade middleware my-redis --to-version 7.2.1 --baseline redis-standalone-v2
saola upgrade mw my-redis --to-version 7.2.1 --wait 5m -n production
```

**注意事项**:
- 若上一次升级尚未完成（annotation 仍存在），命令会拒绝执行并报错。
- `--wait` 通过每 3 秒轮询检查：`middleware.cn/update` annotation 消失且 `status.state == Available`。
- 升级流程由 controller 执行：查找目标版本包 -> 切换 Spec.Baseline -> 更新 Labels -> 删除 annotation -> State: Updating -> Available。

---

#### saola delete middleware

**语法**: `saola delete middleware <name> [flags]`

**说明**: 按名称删除 Middleware 资源。资源不存在时静默跳过。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | Middleware 资源名称 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |
| `--wait` | - | duration | `0`（不等待） | 等待删除完成的超时（每 2 秒轮询） |

**示例**:
```bash
saola delete middleware my-redis
saola delete middleware my-redis -n production
saola delete middleware my-redis --wait 2m
```

---

### 3.3 Operator 管理

使用 `saola get operator`、`saola describe operator`、`saola upgrade operator` 和 `saola delete operator` 管理 Operator；资源类型可简写为 `op`。创建使用 `saola create -f <file>`。

---

#### saola create — MiddlewareOperator

**语法**: `saola create -f <file> [flags]`

**说明**: 从 YAML 文件读取 MiddlewareOperator 清单并在集群中创建。命令会自动从已安装包中查找匹配 `spec.baseline` 的 MiddlewareOperatorBaseline，并补全四个必需的 labels。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--file` | `-f` | string | 必填 | YAML 清单文件路径 |
| `--namespace` | `-n` | string | 空 | 覆盖清单中的命名空间 |

**示例**:
```bash
saola create -f operator.yaml
saola create -f operator.yaml --namespace my-ns
```

**注意事项**:
- 创建前会检查同名资源是否已存在，已存在则报错。
- 若已安装包中未找到匹配的 MiddlewareOperatorBaseline，命令会报错。
- `namespace` 为必填项：必须通过 `--namespace`、manifest 中的 namespace 或全局配置提供。

---

#### saola get operator

**语法**: `saola get operator [name] [flags]`

**说明**: 列出或查询 MiddlewareOperator 资源。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `[name]` | - | 位置参数 | 可选 | 指定则获取单个资源 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |
| `--all-namespaces` | `-A` | bool | `false` | 跨所有命名空间列出 |
| `--output` | `-o` | string | `table` | 输出格式：`table`、`wide`、`yaml`、`json`、`name` |

**示例**:
```bash
saola get operator
saola get operator redis-operator -n my-ns
saola get operator -A -o yaml
```

**注意事项**:
- 表格输出列：`NAME`、`NAMESPACE`、`BASELINE`、`STATE`、`READY`、`RUNTIME`、`AGE`。
- `wide` 格式额外显示 `LABELS` 列。
- 单个资源查询时 namespace 为必填项。

---

#### saola describe operator

**语法**: `saola describe operator <name> [flags]`

**说明**: 打印 MiddlewareOperator 的完整 spec、status、conditions 及每个 operator 的 Deployment 状态。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | MiddlewareOperator 资源名称 |
| `--namespace` | `-n` | string | 空 | 目标命名空间（必填） |

**示例**:
```bash
saola describe operator redis-operator -n my-ns
```

**输出内容**:
- Metadata：Name、Namespace、Labels、Annotations、Created
- Spec：Baseline、PermissionScope、Configurations、PreActions（含 fixed/exposed 属性）
- Status：State、Ready、Runtime、OperatorAvailable、Reason、ObservedGeneration
- Conditions：TYPE、STATUS、REASON、MESSAGE
- OperatorStatus：每个 Deployment 的 Replicas、Ready、Available、Updated、Conditions
- Finalizers

---

#### saola upgrade operator

**语法**: `saola upgrade operator <name> [flags]`

**说明**: 通过设置 `middleware.cn/update` 和 `middleware.cn/baseline` 注解触发 Controller 执行升级。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | MiddlewareOperator 实例名称 |
| `--to-version` | - | string | 必填 | 升级目标版本号 |
| `--baseline` | - | string | 空（保留当前 Baseline） | 升级目标 Baseline 名称 |
| `--namespace` | `-n` | string | 空（必填） | 目标命名空间 |
| `--wait` | - | duration | `0`（不等待） | 等待升级完成的超时时间 |

别名：`op`

**示例**:
```bash
saola upgrade operator redis-op --to-version 1.2.0 -n my-ns
saola upgrade operator redis-op --to-version 1.2.0 --baseline redis-ha -n my-ns
saola upgrade op redis-op --to-version 1.2.0 -n my-ns --wait 5m
```

**注意事项**:
- 若 `middleware.cn/update` annotation 已存在，说明上一次升级尚未完成，命令会拒绝执行。
- Controller 处理时保留 Globe 和 PreActions，清空其余 Spec，切换到新 Baseline。
- `--wait` 通过每 2 秒轮询检查：annotation 消失且 `status.state == Available`；`StateUnavailable` 视为升级失败。

---

#### saola delete operator

**语法**: `saola delete operator <name> [flags]`

**说明**: 按名称删除 MiddlewareOperator。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | MiddlewareOperator 资源名称 |
| `--namespace` | `-n` | string | 空（必填） | 目标命名空间 |
| `--wait` | - | duration | `0`（不等待） | 等待删除完成的超时时间（Finalizer 清理可能需时间） |

**示例**:
```bash
saola delete operator redis-operator --namespace my-ns
saola delete operator redis-operator -n my-ns --wait 2m
```

**注意事项**:
- 资源不存在时静默跳过。
- `--wait` 通过每 2 秒轮询检查对象是否已从集群中完全移除。

---

### 3.4 Baseline 管理

使用 `saola get baseline [name] --package <package>` 查询 Baseline，资源类型可简写为 `bl`。

按名称查询目前仅支持 `middleware` 和 `operator`；Action Baseline 使用 `saola get baseline --package <package> --kind action` 列表查询。

---

#### saola get baseline <name>

**语法**: `saola get baseline <name> [flags]`

**说明**: 从已安装的中间件包中按名称获取单个 MiddlewareBaseline 或 MiddlewareOperatorBaseline。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | Baseline 名称 |
| `--package` | - | string | 必填 | 要查询的包名 |
| `--kind` | - | string | `middleware` | Baseline 类型：`middleware`、`operator` |
| `--output` | `-o` | string | `table` | 输出格式：`table`、`yaml`、`json` |

**示例**:
```bash
saola get baseline default --package redis-v1
saola get baseline default --package redis-v1 --kind operator -o yaml
```

---

#### saola get baseline（列表）

**语法**: `saola get baseline [flags]`（别名：`saola get bl`）

**说明**: 列出指定已安装包中所有 MiddlewareBaseline、MiddlewareOperatorBaseline 或 MiddlewareActionBaseline。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--package` | - | string | 必填 | 要查询的包名 |
| `--kind` | - | string | `middleware` | Baseline 类型：`middleware`、`operator`、`action` |
| `--output` | `-o` | string | `table` | 输出格式：`table`、`yaml`、`json` |

**示例**:
```bash
saola get baseline --package redis-v1
saola get baseline --package redis-v1 --kind middleware
saola get baseline --package redis-v1 --kind operator
saola get baseline --package redis-v1 --kind action
```

**注意事项**:
- 表格输出列（middleware/operator）：`NAME`、`OPERATOR`、`CONFIGURATIONS`、`PREACTIONS`。
- 表格输出列（action）：`NAME`。
- 无结果时输出 "No xxx baselines found."。

---

### 3.5 Action 管理

使用 `saola run` 触发操作，使用 `saola get action` 和 `saola describe action` 查看结果；资源类型可简写为 `act`。

---

#### saola get action

**语法**: `saola get action [name] [flags]`

**说明**: 列出命名空间内的所有 MiddlewareAction，或按名称获取单个资源。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `[name]` | - | 位置参数 | 可选 | 指定则获取单个资源 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |
| `--all-namespaces` | `-A` | bool | `false` | 跨所有命名空间列出 |
| `--output` | `-o` | string | `table` | 输出格式：`table`、`yaml`、`json` |

**示例**:
```bash
saola get action
saola get action my-action-1234567890
saola get action -A
saola get action my-action-1234567890 -o yaml
```

**注意事项**:
- 表格输出列：`NAME`、`MIDDLEWARE`、`BASELINE`、`STATE`、`AGE`。
- `-A` 时额外显示 `NAMESPACE` 列。

---

#### saola describe action

**语法**: `saola describe action <name> [flags]`

**说明**: 按名称获取 MiddlewareAction 并以可读格式展示其完整状态和事件信息。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<name>` | - | 位置参数 | 必填 | MiddlewareAction 名称 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |

**示例**:
```bash
saola describe action my-action-1234567890
saola describe action my-action-1234567890 -n my-namespace
```

**输出内容**:
- Metadata：Name、Namespace、Created
- Spec：Middleware、Baseline、Necessary（格式化的 JSON）
- Status：State、Reason、ObservedGeneration
- Conditions：TYPE、STATUS、REASON、MESSAGE

---

#### saola run

**语法**: `saola run <baseline> --middleware <name> [flags]`

**说明**: 创建一个 MiddlewareAction CR，对指定 Middleware 实例触发一次性运维操作。Action 名称自动生成为 `<baseline>-<unix时间戳>` 以避免冲突。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--middleware` | - | string | 必填 | 关联的 Middleware 实例名 |
| `<baseline>` | - | 位置参数 | 必填 | MiddlewareActionBaseline 名称 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |
| `--params` | - | string array | 空 | `key=value` 格式的操作参数，可逗号分隔或重复使用 |
| `--wait` | - | duration | `0`（不等待） | 等待操作完成的超时时间 |

**示例**:
```bash
saola run redis-backup --middleware my-redis
saola run redis-restore --middleware my-redis --params src=backup-001 --wait 5m
```

**注意事项**:
- `--params` 支持两种写法：
  - 多次指定：`--params key1=val1 --params key2=val2`
  - 逗号分隔：`--params k1=v1,k2=v2`
- 参数会被序列化为 JSON 存入 `spec.necessary` 字段。
- `--wait` 通过每 2 秒轮询检查 `status.state`：`Available` 表示成功，`Unavailable` 表示失败。

---

### 3.6 顶层命令

顶层动词命令提供 kubectl 风格的跨资源入口；包操作还支持 `saola package` / `saola pkg` 写法。

---

#### saola build

`saola build` 等价于 `saola package build`。

**语法**: `saola build <pkg-dir> [--output/-o <path>]`

---

#### saola install

`saola install` 等价于 `saola package install`。

**语法**: `saola install <pkg-dir> [--name <name>] [--wait <duration>] [--dry-run]`

---

#### saola disable

`saola disable package` 等价于 `saola package disable`。

**语法**: `saola disable package <name>`

---

#### saola uninstall

`saola uninstall` 等价于 `saola package uninstall`，也支持 `saola uninstall package <name>`。

**语法**: `saola uninstall <name> [--wait <duration>]` 或 `saola uninstall package <name> [--wait <duration>]`

---

#### saola upgrade

`saola upgrade` 是多功能升级命令，支持包升级和实例升级两种模式：

- **包升级**：`saola upgrade <pkg-dir> [--name <name>] [--wait <duration>]`
  - 等价于 `saola package upgrade`
- **实例升级**：通过子命令路由
  - `saola upgrade middleware <name> --to-version <v> [--baseline <b>] [-n <ns>] [--wait <d>]`
  - `saola upgrade operator <name> --to-version <v> [--baseline <b>] [-n <ns>] [--wait <d>]`
  - 别名：`mw`、`op`

当第一个参数不是已知子命令（`middleware`/`mw`/`operator`/`op`）时，自动走包升级路径。

---

#### saola delete

`saola delete` 通过资源类型子命令路由到各类资源的删除逻辑。

**语法**: `saola delete <resource-type> <name> [flags]`

**支持的资源类型**:

| 类型 | 别名 | 说明 |
|------|------|------|
| `middleware` | `mw` | 删除 Middleware 资源 |
| `operator` | `op` | 删除 MiddlewareOperator 资源 |

**额外 Flag**:
- `--dry-run`：仅打印将要删除的资源，不实际执行
- `--namespace` / `-n`：目标命名空间
- `--wait`：等待删除完成的超时时间

**示例**:
```bash
saola delete middleware my-redis
saola delete mw my-redis -n production
saola delete operator redis-operator -n my-ns --wait 2m
saola delete middleware my-redis --dry-run
```

---

#### saola get

`saola get` 是统一的资源查询入口，通过资源类型子命令路由。

**语法**: `saola get <resource-type> [name] [flags]`

**支持的资源类型**:

| 类型 | 别名 | 说明 |
|------|------|------|
| `middleware` | `mw` | Middleware 资源 |
| `operator` | `op` | MiddlewareOperator 资源 |
| `action` | `act` | MiddlewareAction 资源 |
| `baseline` | `bl` | Baseline 资源（需 `--package`） |
| `package` | `pkg` | 已安装的中间件包 |
| `all` | - | 聚合输出 middleware、operator 和 action |

**示例**:
```bash
saola get middleware
saola get mw my-redis -o yaml
saola get operator -A
saola get action -n my-ns
saola get baseline --package redis-v1 --kind operator
saola get package --component redis
saola get package redis-v1
saola get all -n my-ns
```

**注意事项**:
- `saola get package [name]` 只读取包 Secret 元数据；无 name 时列出包，可用 `--component` 和 `--version` 过滤。查看归档文件内容使用 `saola inspect <name>`。
- `saola get baseline [name]`：有 name 时查询单个 Baseline，无 name 时列出包内 Baseline。
- `saola get all` 按顺序输出 Middlewares、Operators、Actions，每类之间用分隔标题区分。

---

#### saola describe

`saola describe` 通过资源类型子命令显示资源详情。

**语法**: `saola describe <resource-type> <name> [flags]`

**支持的资源类型**:

| 类型 | 别名 |
|------|------|
| `middleware` | `mw` |
| `operator` | `op` |
| `action` | `act` |

**示例**:
```bash
saola describe middleware my-redis
saola describe mw my-redis -n production
saola describe operator redis-operator -n my-ns
saola describe action my-action-1234567890
```

---

#### saola inspect

`saola inspect` 等价于 `saola package inspect`。

**语法**: `saola inspect <name> [-o table|yaml|json]`

---

#### saola create（含交互式模式）

`saola create` 支持两种模式：

- **文件模式**：`saola create -f <file> [-n <namespace>] [--dry-run]`
  - 自动检测 YAML 中的 `kind` 字段，路由到 Middleware 或 MiddlewareOperator 创建逻辑
  - 支持的 kind：`Middleware`、`MiddlewareOperator`
- **交互式模式**：`saola create`（不带 `-f`）
  - 进入终端引导式创建流程（详见第 7 章）

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--file` | `-f` | string | 空 | YAML 文件路径；为空时进入交互模式 |
| `--namespace` | `-n` | string | 空 | 覆盖清单中的命名空间 |
| `--dry-run` | - | bool | `false` | 仅打印将创建的资源（仅文件模式） |

**示例**:
```bash
# 交互式创建
saola create

# 从文件创建
saola create -f middleware.yaml
saola create -f operator.yaml -n production
saola create -f middleware.yaml --dry-run
```

文件模式的 `create --dry-run` 只读取 YAML 并打印 kind/name，不执行服务端校验或包与 Baseline 匹配。

---

#### saola run

`saola run` 触发一次性运维操作，baseline 名称作为位置参数传入。

**语法**: `saola run <baseline> --middleware <name> [flags]`

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `<baseline>` | - | 位置参数 | 必填 | MiddlewareActionBaseline 名称 |
| `--middleware` | - | string | 必填 | 关联的 Middleware 实例名 |
| `--namespace` | `-n` | string | 空 | 目标命名空间 |
| `--params` | - | string array | 空 | `key=value` 格式的操作参数 |
| `--wait` | - | duration | `0` | 等待操作完成 |

**示例**:
```bash
saola run redis-backup --middleware my-redis
saola run redis-restore --middleware my-redis --params src=backup-001 --wait 5m
```

---

#### saola version

**语法**: `saola version [-o json|yaml]`

**说明**: 打印 saola-cli 的版本号、构建时间和 Git 提交信息。

**参数/Flags**:

| 名称 | 缩写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--output` | `-o` | string | 空（人类可读格式） | 输出格式：`json`、`yaml` |

**示例**:
```bash
saola version
saola version -o json
```

**输出示例**（默认格式）:
```
Version: v1.0.0
Git Commit: abc1234
Build Date: 2026-01-01T00:00:00Z
```

---

### 3.7 中间件包镜像导出与导入

`saola images export <pkg-dir>` 从本地中间件包目录发现镜像引用并导出镜像归档；`saola images import <archive>` 将归档中的镜像推送到目标仓库。两者不需要连接 Kubernetes 集群。`saola build` 生成的包文件不包含这些镜像，离线交付时需要分别准备包和镜像归档。

#### 导出并交付到离线仓库

安装 `skopeo`，并使用 `skopeo login` 登录需要认证的源仓库。以下仓库地址均为示例，请替换为实际地址：

```bash
skopeo login registry.example.com

# 查看包中发现的镜像和候选来源，不探测仓库、不下载镜像
saola images export ./redis -r registry.example.com/middleware --dry-run

# 按声明顺序尝试候选仓库，默认保留镜像的多架构清单
saola images export ./redis \
  -r registry.example.com/middleware \
  -r registry-backup.example.com/middleware \
  -o redis-images.tar
```

成功后同时交付 `redis-images.tar` 和 `redis-images.tar.lock.json`。锁定清单保存导入所需的镜像名称；重命名或单独存放时，使用 `--lock-file` 指定对应清单。

在能够访问目标仓库的机器上安装 `skopeo`，复制上述两个文件，然后执行：

```bash
skopeo login registry.internal.example.com
saola images import redis-images.tar \
  -r registry.internal.example.com/middleware --dry-run
saola images import redis-images.tar \
  -r registry.internal.example.com/middleware
```

导入目标为 `<目标仓库>/<镜像名>:<tag>`，例如 `registry.internal.example.com/middleware/redis:7.2.4`。`images import` 执行仓库推送；加载到本机 Docker 请使用下方的 Docker 归档方式。导入不会修改中间件包中的镜像配置，部署时仍需配置对应的目标仓库。

#### 选择归档格式

| `--format` | 导出依赖 | 用途与架构行为 |
|------------|----------|----------------|
| `skopeo`（默认） | skopeo | 供 `saola images import` 使用的 OCI 归档；默认 `--platform all --multi-arch=true` 保留多架构清单 |
| `docker` | Docker 或 nerdctl | 单平台归档，使用 `docker load` / `nerdctl load`；显式指定 `--platform` |
| `docker-multi` | skopeo | 根级 OCI 归档，默认要求 `linux/amd64,linux/arm64`；使用启用了 containerd image store 的现代 Docker 加载 |

```bash
# 传统 Docker：按平台分别导出
saola images export ./redis -r registry.example.com/middleware \
  --format docker --platform linux/amd64 -o redis-amd64.tar
docker load -i redis-amd64.tar

# 现代 Docker：同时交付 AMD64 和 ARM64
saola images export ./redis -r registry.example.com/middleware \
  --format docker-multi --platforms linux/amd64,linux/arm64 \
  -o redis-multi.tar
docker load -i redis-multi.tar
```

`docker-multi` 会检查每个镜像是否包含所有指定平台，候选仓库缺架构时尝试下一个。默认情况下，存在无法满足平台要求的镜像就会失败。`docker` 和 `docker-multi` 归档不适用于当前的 `saola images import`；需要推送离线仓库时使用默认 `skopeo` 格式。

#### 导出参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-r, --repository` | 必填 | 候选仓库，可重复或用逗号分隔，按顺序探测 |
| `-o, --output` | `<name>-<version>-images.tar` | 镜像归档路径；此处不是查询命令的输出格式 |
| `--lock-file` | `<output>.lock.json` | 锁定清单输出路径 |
| `--format` | `skopeo` | `skopeo`、`docker` 或 `docker-multi` |
| `--platform` | `all` | `skopeo` / `docker` 的平台选择，如 `linux/amd64` |
| `--platforms` | `linux/amd64,linux/arm64` | `docker-multi` 必须包含的平台 |
| `--multi-arch` | `true` | skopeo 导出时保留多架构清单 |
| `--insecure` | `false` | 跳过仓库 TLS 校验；docker 格式使用此参数时也需要 skopeo |
| `--skip-missing` | `false` | 允许仅导出已解析的镜像，产物可能不完整；全部未命中仍失败 |
| `--dry-run` | `false` | 仅列出候选，不检查仓库是否存在镜像，也不生成归档和锁定清单 |
| `--timeout` | `30s` | 单个镜像探测超时 |

导出使用实际后端工具的登录凭据，没有 `--creds` 参数。使用 Docker 或 nerdctl 后端时，先通过对应工具登录源仓库。`--insecure` 不会跳过身份认证。

#### 导入参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-r, --repository` | 必填 | 单个目标仓库路径 |
| `--lock-file` | `<archive>.lock.json` | 与归档匹配的镜像锁定清单，缺失时报错 |
| `--creds` | 空 | 目标仓库凭据，格式 `user:password`；也可预先执行 `skopeo login` |
| `--platform` | `all` | 导入平台，如 `linux/amd64` |
| `--multi-arch` | `true` | 推送时保留多架构清单 |
| `--insecure` | `false` | 跳过目标仓库 TLS 校验 |
| `--dry-run` | `false` | 从已有归档及锁定清单列出目标镜像，不推送 |

#### 镜像发现范围

当前发现逻辑基于 `metadata.app.version` 和 necessary 参数默认值渲染包内容，不能穷举非默认条件分支，也没有 `--set` / `--values` 参数。离线交付前需另外盘点可选功能、条件分支、升级任务和 CRD 默认镜像；`--dry-run` 输出不等于所有场景的完整镜像清单。

---

### 3.8 本地包校验

`saola validate <pkg-dir>`（等价于 `saola package validate`）检查 `metadata.yaml`、YAML 文档、OpenSaola CR 结构和 MiddlewareConfiguration 模板渲染结果，不连接集群。校验失败返回非零退出码；通过时输出文件、文档和已渲染模板的统计。

```bash
saola validate ./redis
saola package validate ./redis -o json
```

`-o, --output` 支持 `table`（默认摘要）、`yaml`、`json`。`saola build` 和 `saola upgrade <pkg-dir>` 会先执行包校验；安装前也可以显式运行上述命令。此校验不代替实际部署和中间件功能验收。

### 3.9 命令帮助与 Shell 补全

```bash
saola --help
saola help images export
saola --lang en images export --help

# 为当前 Bash 会话启用补全
source <(saola completion bash)

# 为当前 Zsh 会话启用补全
# 首次使用时先初始化 Zsh 补全
autoload -Uz compinit && compinit
source <(saola completion zsh)
```

还支持 `saola completion fish` 和 `saola completion powershell`。使用 `saola completion <shell> --help` 查看对应 Shell 的持久化安装说明。命令与参数以当前二进制的 `--help` 为准。

---

## 4. 包（Package）规范

### 4.1 TAR 包格式和目录结构

saola-cli 将本地目录打包为 zstd 压缩的 TAR 归档。TAR 内部根目录前缀为 `<name>-<version>/`，以便 OpenSaola 的 `ReadTarInfo` 正确剥离首层路径。

**预期目录结构**：

```
<pkg-dir>/
  metadata.yaml          # 包元数据（必需）
  baselines/             # Middleware/Operator/Action Baseline 定义
  configurations/        # 配置模板
  actions/               # Action 模板
  crds/                  # CRD 定义
  manifests/             # 平台前端配置文件
```

`manifests/` 目录包含平台前端使用的配置文件（如参数表单定义 `parameters.yaml`、国际化 `i18n.yaml`、版本规则 `middlewareversionrule.yaml`、隐藏菜单 `hiddenmenus.yaml`），该目录内容不被 OpenSaola 直接处理，由平台前端组件消费。

**打包规则**：
- 隐藏文件和目录（以 `.` 开头，如 `.git`、`.DS_Store`）会被自动跳过。
- 仅打包普通文件，目录本身不存入 TAR。
- 路径分隔符统一为 `/`（跨平台兼容）。
- 使用 `packages.Compress()` 进行 zstd 压缩。

### 4.2 metadata.yaml 要求

```yaml
name: redis                      # 必填：包名称
version: "1.0.0"                 # 必填：包版本
app:
  version: ["7.2.0", "7.2.1"]   # 支持的应用版本列表
  deprecatedVersion: []          # 已废弃的版本列表
owner: "team-a"                  # 包所有者
type: "cache"                    # 包类型（合法值：db / mq / cache / search / storage）
description: "Redis package"     # 包描述
```

**必填字段**：`name`、`version`。缺少时 `PackDir()` 会返回错误。

### 4.3 包状态机

包通过 Secret 上的 Labels 和 Annotations 管理状态：

```
[安装请求]
    |
    v
Secret 创建 (enabled=false, install=true)
    |
    v (OpenSaola 处理)
安装成功 → enabled=true, install annotation 移除
安装失败 → enabled=false, installError annotation 设置
    |
[卸载请求]
    |
    v
CLI 检查 Middleware / MiddlewareOperator 引用
    |
    v
添加 package-secret-cleanup finalizer 并删除 Secret
    |
    v (Secret Terminating，OpenSaola 二次检查并处理)
无引用 → 清理包资源 → 删除 MiddlewarePackage → 移除 finalizer → Secret 删除
仍被引用 → 写入 uninstallError，保留 finalizer
```

---

## 5. Secret 结构

### 5.1 包安装后生成的 Secret

`saola install` / `saola package install` 创建的 Secret 结构如下：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>-<version>         # 默认，可通过 --name 覆盖
  namespace: middleware-operator  # pkg-namespace
  labels:
    middleware.cn/project: OpenSaola
    middleware.cn/component: <metadata.name>
    middleware.cn/packageversion: <metadata.version>
    middleware.cn/packagename: <secret-name>
    middleware.cn/enabled: "false"
  annotations:
    middleware.cn/install: "true"
immutable: true
data:
  package: <zstd-compressed-tar-bytes>
```

### 5.2 Secret Labels 和 Annotations

**Labels**:

| Label Key | 值 | 说明 |
|-----------|------|------|
| `middleware.cn/project` | `OpenSaola` | 标识该 Secret 属于 OpenSaola 系统 |
| `middleware.cn/component` | `<metadata.name>` | 中间件组件名称（如 `redis`） |
| `middleware.cn/packageversion` | `<metadata.version>` | 包版本号 |
| `middleware.cn/packagename` | `<secret-name>` | 包 Secret 名称 |
| `middleware.cn/enabled` | `"true"` / `"false"` | 包是否已成功安装并启用 |

**Annotations**:

| Annotation Key | 值 | 说明 |
|----------------|------|------|
| `middleware.cn/install` | `"true"` | 触发安装 |
| `middleware.cn/uninstall` | `"true"` | 触发兼容的软卸载路径 |
| `middleware.cn/installError` | 错误信息 | 安装失败时由 operator 设置 |
| `middleware.cn/uninstallError` | 错误信息 | 卸载失败时由 operator 设置 |

**Finalizers**:

| Finalizer | 说明 |
|-----------|------|
| `middleware.cn/package-secret-cleanup` | 包 Secret 删除保护；OpenSaola 清理包资源并删除 MiddlewarePackage 后移除 |

### 5.3 Secret data 字段

| Key | 说明 |
|-----|------|
| `package`（`packages.Release` 常量，值为 `"package"`） | zstd 压缩的 TAR 归档字节内容 |

Secret 标记为 `immutable: true`，因此升级时必须删除后重建。

---

## 6. 完整工作流示例

### 6.1 从零开始的完整流程

```bash
# 1. 构建包（离线场景可选）
saola build ./my-redis-pkg --output ./dist/redis-1.0.0.pkg

# 2. 安装包到集群
saola install ./my-redis-pkg --wait 5m

# 3. 查看已安装的包
saola get package

# 4. 查看包中的 baseline
saola get baseline --package redis-1.0.0

# 5. 创建 MiddlewareOperator（若需要）
saola create -f operator.yaml -n my-ns

# 6. 创建 Middleware 实例
saola create -f middleware.yaml -n my-ns
# 或使用交互式模式
saola create

# 7. 查看资源状态
saola get all -n my-ns

# 8. 查看详情
saola describe middleware my-redis -n my-ns
```

### 6.2 包生命周期

```bash
# 安装
saola install ./redis-pkg --wait 5m

# 查看
saola get package
saola inspect redis-1.0.0

# 升级
saola upgrade ./redis-pkg-v2 --wait 5m

# 卸载
saola uninstall redis-1.0.0 --wait 5m
```

### 6.3 中间件实例生命周期

```bash
# 创建
saola create -f middleware.yaml -n production

# 查看状态
saola get middleware -n production
saola describe middleware my-redis -n production

# 版本升级
saola upgrade middleware my-redis --to-version 2.0.0 -n production --wait 5m

# 执行运维操作
saola run redis-backup --middleware my-redis -n production --wait 5m

# 删除
saola delete middleware my-redis -n production --wait 2m
```

### 6.4 Operator 生命周期

```bash
# 创建
saola create -f operator.yaml -n production

# 查看状态
saola get operator -n production
saola describe operator redis-operator -n production

# 升级
saola upgrade operator redis-operator --to-version 2.0.0 -n production --wait 5m

# 删除
saola delete operator redis-operator -n production --wait 2m
```

---

### 6.5 中间件包与镜像离线交付

本例将本地 `./redis-pkg` 及其镜像交付到离线环境。源仓库为 `registry.example.com/middleware`，目标为 `registry.internal.example.com/middleware`，请替换为实际地址。源机器能拉取镜像；目标机器能访问内网镜像仓库及已安装 OpenSaola 的 Kubernetes 集群。两台机器均安装 saola 和 skopeo。

**第一步：源环境检查并导出。**

```bash
saola validate ./redis-pkg
skopeo login registry.example.com
saola images export ./redis-pkg -r registry.example.com/middleware --dry-run
saola images export ./redis-pkg \
  -r registry.example.com/middleware -o redis-images.tar
```

`--dry-run` 用于检查候选引用，真正的镜像探测和下载发生在下一条导出命令。另行核对可选功能和非默认条件分支的镜像，发现范围见第 3.7 节。

**第二步：传递交付内容。** 将以下内容一起复制到目标机器的工作目录：

- `redis-pkg/`：完整中间件包目录，供 `saola install` 使用。
- `redis-images.tar`：镜像归档。
- `redis-images.tar.lock.json`：与归档匹配的镜像锁定清单。

当前 `install` 接受包目录。若另用 `saola build` 生成 `.pkg` 作为交付制品，不要直接把 `.pkg` 路径当作安装目录传入。

**第三步：目标环境导入镜像。**

```bash
skopeo login registry.internal.example.com
saola images import redis-images.tar \
  -r registry.internal.example.com/middleware --dry-run
saola images import redis-images.tar \
  -r registry.internal.example.com/middleware
```

**第四步：配置镜像仓库并安装包。** 按该中间件包的 README 配置实际使用的镜像仓库，使部署引用与上一步导入目标一致；`images import` 不会自动改写包或实例配置。确认 kubeconfig 指向目标集群后执行：

```bash
saola validate ./redis-pkg
saola install ./redis-pkg --name redis-offline \
  --pkg-namespace middleware-operator --wait 5m
saola get package redis-offline
saola get baseline --package redis-offline --kind operator
saola get baseline --package redis-offline --kind middleware

# 交互式选择该包的 Baseline，创建所需 Operator 和中间件实例
saola create
saola get all -n my-ns
```

`redis-offline` 是本例显式指定的包 Secret 名称；`my-ns` 替换为创建实例时选择的命名空间。需要独立 Operator 的包先创建 Operator，再创建中间件实例。包安装成功后，还需检查实例状态并按中间件协议验证可用性。

---

## 7. 交互式创建详解

### 7.1 交互流程

运行 `saola create`（不带 `-f` 参数）进入交互式创建模式，使用 charmbracelet/huh 库实现终端表单。

**Middleware 创建流程**：

1. **选择资源类型**：Middleware 或 MiddlewareOperator
2. **选择中间件组件**：若已安装包中包含多个组件（如 Redis、PostgreSQL），先选择组件类型
3. **选择 Baseline**：在选定组件下选择具体的 MiddlewareBaseline
4. **填写元数据**：输入实例名称和命名空间
5. **填写 necessary 参数**：根据 baseline 的 `spec.necessary` JSON Schema 动态生成表单
6. **YAML 预览**：显示将要创建的完整 Middleware YAML
7. **确认创建**：确认后实际创建资源

**MiddlewareOperator 创建流程**：

1. **选择资源类型**：选择 MiddlewareOperator
2. **选择中间件组件**：同上
3. **选择 OperatorBaseline**：在选定组件下选择 MiddlewareOperatorBaseline
4. **填写元数据**：输入实例名称和命名空间
5. **配置 Globe**：若 baseline 包含 `spec.globe`（镜像仓库配置），逐字段展示可编辑输入框，预填 baseline 默认值；仅当用户修改了至少一个值时才写入 `mo.spec.globe`
6. **YAML 预览**：显示完整的 MiddlewareOperator YAML
7. **确认创建**

### 7.2 JSON Schema 支持

交互式模式从 baseline 的 `spec.necessary` 字段解析 JSON Schema，支持以下字段类型：

| 类型 | 表单组件 | 验证规则 |
|------|----------|----------|
| `string` | 文本输入框 | `required` 校验、`pattern` 正则校验 |
| `int` | 文本输入框 | 整数校验、`min`/`max` 范围校验 |
| `password` | 密码输入框（隐藏输入） | `required` 校验、多条 `patterns` 正则校验 |
| `enum` | 下拉选择框 | 选项来自 `options` 字段（逗号分隔） |
| `version` | 下拉选择框 / 文本输入框 | 选项来自包 metadata 的 `app.version` 列表；无列表时回退为文本输入 |
| `storageClass` | 下拉选择框 / 文本输入框 | 选项来自集群的 StorageClass 列表；无法获取时回退为文本输入 |

**FieldSchema 结构**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 字段类型（必填，为空则跳过） |
| `label` | string | 人类可读的显示名称 |
| `required` | bool | 是否为必填项 |
| `default` | interface{} | 默认值 |
| `description` | string | 补充说明 |
| `placeholder` | string | 空输入框的提示文本 |
| `pattern` | string | 正则验证（string 类型使用） |
| `min` | float64 | 数值下界（int 类型使用） |
| `max` | float64 | 数值上界（int 类型使用） |
| `options` | string | 逗号分隔的允许值列表（enum 类型使用） |
| `patterns` | []PatternRule | 多条验证规则（password 类型使用） |

### 7.3 参数验证

- **必填校验**：`required=true` 的字段不允许空白输入。
- **正则校验**：`pattern` 或 `patterns` 中定义的正则表达式用于输入验证，不匹配时显示描述信息。
- **范围校验**：`int` 类型支持 `min` 和 `max`，超出范围时提示最小/最大值。
- **枚举校验**：`enum` 类型仅允许从 `options` 中选择。

**值收集与嵌套**：

用户输入的值按点分隔路径（如 `resource.postgresql.limits.cpu`）收集为扁平 map，然后通过 `BuildNecessaryValues()` 转换为嵌套结构：

```
{"resource.postgresql.limits.cpu": "1"}
  -->
{"resource": {"postgresql": {"limits": {"cpu": "1"}}}}
```

---

## 8. 输出格式

### 8.1 Table 格式

默认输出格式。使用 `text/tabwriter` 实现对齐的表格。支持三种输入数据类型：

- `[][]string`：首行为表头，后续为数据行
- `[]map[string]string`：key 作为列名
- 任意 struct 切片：导出字段名作为列名（大写），值通过 `fmt.Sprint` 格式化

`wide` 格式是 `table` 的变体，调用方传入包含额外字段（如 LABELS）的行结构体。

### 8.2 JSON 格式

使用 `encoding/json` 的 `Encoder`，缩进 2 空格。

### 8.3 YAML 格式

使用 `gopkg.in/yaml.v3` 的 `Encoder`，缩进 2 空格。

### 8.4 Name 格式

输出 `type/name` 格式，每行一条。类似 `kubectl get -o name`。

- 通过 NamePrinter 实现，需要调用方设置 `ResourceType` 字段。
- 支持通过反射从 struct 中提取 `Name` 或 `NAME` 字段，也支持嵌入的 `ObjectMeta.Name`。

---

## 9. 配置与环境

### 9.1 配置文件

saola-cli 当前不使用独立配置文件。所有配置通过命令行 flag 和环境变量传入。

`Config` 结构体包含以下字段：

| 字段 | 来源（优先级从高到低） | 默认值 |
|------|----------------------|--------|
| `Kubeconfig` | `--kubeconfig` -> `$KUBECONFIG` | 空（使用 `~/.kube/config`） |
| `Context` | `--context` | 空（使用 kubeconfig 默认 context） |
| `Namespace` | `--namespace` -> `$SAOLA_NAMESPACE` | 空（命令层面回退到 `default`） |
| `PkgNamespace` | `--pkg-namespace` -> `$SAOLA_PKG_NAMESPACE` | `middleware-operator` |
| `LogLevel` | `--log-level` | `info` |
| `NoColor` | `--no-color` | `false` |

### 9.2 环境变量

参见 [2.5 环境变量](#25-环境变量)。

---

## 10. 与其他项目的数据流

### 10.1 saola-cli --> dataservice-baseline（消费包）

saola-cli 的 `build` / `install` 命令将 dataservice-baseline 项目构建产出的本地目录打包为 TAR 归档。包目录中包含：

- `metadata.yaml`：包元数据
- `baselines/`：MiddlewareBaseline、MiddlewareOperatorBaseline、MiddlewareActionBaseline 的 YAML/JSON 定义
- `configurations/`：CUE/Go 配置模板
- `actions/`：Action 执行模板
- `crds/`：CRD 定义文件

### 10.2 saola-cli --> OpenSaola（创建 K8s 资源）

saola-cli 在 Kubernetes 集群中创建以下资源，由 OpenSaola 消费：

| 资源类型 | 创建方式 | OpenSaola 行为 |
|----------|----------|-------------------|
| Secret（包） | `saola install` / `saola upgrade` | 监听 `install` annotation，解压 TAR，加载 baseline/config/action 模板 |
| Middleware CR | `saola create` | Reconcile 循环：查找包 -> 渲染模板 -> 部署底层资源 -> 状态上报 |
| MiddlewareOperator CR | `saola create` | Reconcile 循环：部署 operator deployment -> 就绪检查 -> 状态上报 |
| MiddlewareAction CR | `saola run` | Reconcile 循环：执行一次性操作 -> 状态上报 |

### 10.3 完整数据流图

```
+------------------------+
| dataservice-baseline   |
| (CUE/Go 模板、         |
|  Baseline 定义)        |
+----------+-------------+
           |
           | 构建产出
           v
+----------+-------------+
| 本地包目录              |
| metadata.yaml          |
| baselines/             |
| configurations/        |
| actions/               |
| crds/                  |
+----------+-------------+
           |
           | saola build / install
           v
+----------+-------------+
| K8s Secret             |
| (zstd TAR in data)     |  <--- pkg-namespace (middleware-operator)
| Labels + Annotations   |
+----------+-------------+
           |
           | OpenSaola Watch
           v
+----------+-------------+
| OpenSaola          |
| Package Service        |
| (解压、解析、缓存)      |
+----------+-------------+
           |
           | saola create / middleware create / operator create
           v
+----------+---+---+-----+
| Middleware   | MiddlewareOperator  | MiddlewareAction  |
| CR           | CR                  | CR                |
+--------------+-----+---------+----+-------------------+
                     |         |        |
                     v         v        v
              OpenSaola Reconcile Controllers
              (部署底层资源、状态管理、升级、删除)
```
