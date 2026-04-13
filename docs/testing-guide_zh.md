# Saola CLI 测试指南

[English](testing-guide.md) | **中文**

本指南涵盖 saola-cli 的单元测试、构建、端到端（E2E）测试和 CI 检查。

## 1. 单元测试

```bash
# 运行所有单元测试
make test

# 运行指定包的测试（带详细输出）
go test ./internal/packager/... -v
go test ./internal/tarutil/... -v -run TestReadTarInfo
go test ./internal/printer/... -v
go test ./internal/lang/... -v
```

### 包覆盖率矩阵

| 包 | 有测试 | 说明 |
|---|--------|------|
| `internal/packager` | 是 | TAR/zstd 打包逻辑 |
| `internal/tarutil` | 是 | TAR 读写辅助函数 |
| `internal/printer` | 是 | 输出格式化（table/yaml/json） |
| `internal/lang` | 是 | 中英双语消息查找 |
| `internal/config` | 是 | 配置管理 |
| `internal/cmdutil` | 是 | 命令工具函数 |
| `internal/k8s` | 是 | Kubernetes 辅助函数 |
| `internal/version` | 是 | 版本信息 |
| `internal/waiter` | 是 | 异步等待逻辑 |
| `internal/cmd/pkgcmd` | 是 | 包管理命令 |
| `internal/cmd/operator` | 是 | Operator 命令 |
| `internal/cmd/middleware` | 是 | Middleware 命令 |
| `internal/cmd/create` | 是 | 资源创建 |
| `internal/cmd/baseline` | 是 | Baseline 查询 |
| `internal/cmd/action` | 是 | Action 命令 |
| `internal/client` | 是 | Kubernetes client 封装 |
| `internal/app` | 是 | 根命令注册 |
| `internal/consts` | 是 | 常量定义 |
| `internal/packages` | 是 | 包定义 |
| `internal/cmd/build` | 是 | 构建命令 |
| `internal/cmd/delete` | 是 | 删除命令 |
| `internal/cmd/describe` | 是 | 详情命令 |
| `internal/cmd/get` | 是 | 列表命令 |
| `internal/cmd/inspect` | 是 | 检查命令 |
| `internal/cmd/install` | 是 | 安装命令 |
| `internal/cmd/uninstall` | 是 | 卸载命令 |
| `internal/cmd/upgrade` | 是 | 升级命令 |
| `internal/cmd/resource` | 是 | 资源命令 |
| `internal/cmd/run` | 是 | 运行命令 |
| `internal/cmd/version` | 是 | 版本命令 |
| `cmd/saola` | 否 | 主入口 |

## 2. 构建

```bash
# 构建二进制文件
make build

# 验证构建结果
./bin/saola version
```

预期输出：

```
Version: v0.1.0
Git Commit: <commit-hash>
Build Date: <timestamp>
```

## 3. E2E 测试流程

### 前置条件

1. **部署 opensaola operator**：
   ```bash
   cd ../opensaola
   make docker-build IMG=opensaola:test
   make install
   make deploy IMG=opensaola:test
   kubectl wait --for=condition=available --timeout=120s deploy/opensaola-controller-manager -n opensaola-system
   ```
   详细说明请参见 [opensaola 测试指南](https://gitee.com/opensaola/opensaola/blob/master/docs/testing-guide_zh.md#构建和部署进行手动测试)。

2. **中间件包目录**（例如 `../dataservice-baseline/clickhouse`）

3. **kubectl** 已配置集群访问权限

### 环境准备

```bash
SAOLA=./bin/saola
PKG_DIR=/path/to/dataservice-baseline/clickhouse
PKG_NS=middleware-operator
NS=e2e-test

# 构建 CLI
make build
```

### 步骤 1：创建命名空间

```bash
kubectl create namespace $PKG_NS --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace $NS --dry-run=client -o yaml | kubectl apply -f -
```

### 步骤 2：构建包

```bash
$SAOLA build $PKG_DIR
```

### 步骤 3：安装包

```bash
$SAOLA install $PKG_DIR --pkg-namespace $PKG_NS
```

### 步骤 4：验证包安装

```bash
# 列出已安装的包
$SAOLA get package --pkg-namespace $PKG_NS

# 检查包内容
$SAOLA inspect <pkg-name> --pkg-namespace $PKG_NS

# 列出包中的 Baseline
$SAOLA get baseline --package <pkg-name> --pkg-namespace $PKG_NS

# 按类型过滤 Baseline
$SAOLA get baseline --package <pkg-name> --kind operator --pkg-namespace $PKG_NS
```

将 `<pkg-name>` 替换为 `get package` 输出中的实际包名（例如 `clickhouse-0.25.5-1.0.0`）。

### 步骤 5：安装 ClickHouse CRD

```bash
kubectl apply -f $PKG_DIR/crds/
```

### 步骤 6：创建 MiddlewareOperator

```bash
kubectl apply -f docs/e2e-samples/clickhouse-operator.yaml
```

### 步骤 7：验证 Operator

```bash
# 监视 Operator 状态
kubectl get mo -n $NS -w

# 通过 saola 列出 Operator
$SAOLA get operator -n $NS

# 查看 Operator 详情
$SAOLA describe operator clickhouse-operator -n $NS

# 验证 Operator Deployment 和 Pod
kubectl get deploy -n $NS
kubectl get pods -n $NS
```

等待 Operator 进入 `Running` 状态。

### 步骤 8：创建 Middleware 实例

```bash
kubectl apply -f docs/e2e-samples/clickhouse-middleware.yaml
```

### 步骤 9：验证 Middleware

```bash
# 监视 Middleware 状态
kubectl get mid -n $NS -w

# 通过 saola 列出 Middleware
$SAOLA get middleware -n $NS

# 查看 Middleware 详情
$SAOLA describe middleware my-clickhouse -n $NS

# 验证 Pod 和 ClickHouseInstallation CR
kubectl get pods -n $NS
kubectl get chi -n $NS
```

等待 Middleware 进入 `Running` 状态。

### 步骤 10：测试输出格式

```bash
# 聚合视图
$SAOLA get all -n $NS

# YAML 输出
$SAOLA get middleware -n $NS -o yaml

# JSON 输出
$SAOLA get middleware -n $NS -o json

# Wide 输出
$SAOLA get operator -n $NS -o wide
```

### 步骤 11：清理

```bash
# 删除 Middleware 实例
$SAOLA delete middleware my-clickhouse -n $NS

# 删除 Operator
$SAOLA delete operator clickhouse-operator -n $NS

# 卸载包
$SAOLA uninstall <pkg-name> --pkg-namespace $PKG_NS

# 删除测试命名空间
kubectl delete namespace $NS
```

## 4. CI 检查

提交 PR 前请运行以下检查：

```bash
make lint     # go vet 静态分析
make test     # 单元测试
make build    # 验证编译
```

三个命令必须全部通过，零错误。

## 5. 故障排查

| 现象 | 原因 | 解决方法 |
|------|------|---------|
| `make build` 报缺少模块 | 依赖未拉取 | 先运行 `make tidy` |
| `saola get package` 返回空 | 包未安装或命名空间不对 | 检查 `--pkg-namespace` 参数 |
| Operator 卡在 `Pending` | CRD 未安装 | 运行 `kubectl apply -f $PKG_DIR/crds/` |
| Middleware 卡在 `Pending` | Operator 未就绪 | 等待 Operator 进入 `Running` 状态 |
| `connection refused` 错误 | kubeconfig 未配置 | 检查 `KUBECONFIG` 环境变量或 `--kubeconfig` 参数 |
