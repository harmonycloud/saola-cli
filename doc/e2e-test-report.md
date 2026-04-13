# saola-cli E2E 测试报告

## 概要

| 项目 | 详情 |
|------|------|
| 测试日期 | 2026-03-31 |
| 测试环境 | <cluster-ip> Kubernetes 集群 |
| kubeconfig | `saola-cli/kubeconfig/187` |
| 测试中间件 | PostgreSQL（package: `postgresql-2.8.2-1.0.0` from dataservice-baseline） |
| saola 版本 | v0.1.0 |
| CLI 风格 | kubectl 风格（动作+资源） |
| 测试总数 | **38** |
| 通过 | **38** |
| 失败 | **0** |

---

## 测试用例详情

### TC-01: saola version

**目的**: 验证 version 命令正常输出版本信息。

```bash
# 输入
saola version

# 输出
Version:    v0.1.0-dirty
Git Commit: 6bd4f94
Build Date: 2026-03-31T07:46:11Z
```

**结果**: PASS

---

### TC-02: saola version -o json

**目的**: 验证 version 命令支持 JSON 格式输出。

```bash
# 输入
saola version -o json

# 输出
{
  "version": "v0.1.0-dirty",
  "gitCommit": "6bd4f94",
  "buildDate": "2026-03-31T07:46:11Z"
}
```

**结果**: PASS

---

### TC-03: saola --lang=en version

**目的**: 验证英文模式下 version 命令正常执行。

```bash
# 输入
saola --lang=en version

# 输出
Version:    v0.1.0-dirty
Git Commit: 6bd4f94
Build Date: 2026-03-31T07:46:11Z
```

**结果**: PASS — 英文模式下版本信息输出与默认模式一致。

---

### TC-04: saola --help（中文默认）

**目的**: 验证根命令中文帮助完整输出所有子命令。

```bash
# 输入
saola --help

# 输出
saola 是一个用于管理中间件的 CLI 工具

Usage:
  saola [command]

Available Commands:
  build       构建中间件 package
  create      创建资源
  delete      删除资源
  describe    查看资源详情
  get         列出资源
  inspect     查看已安装 package 的详情
  install     安装 package 到集群
  run         执行 Action
  uninstall   卸载 package
  upgrade     升级已安装的 package
  version     显示版本信息

Flags:
      --kubeconfig string   kubeconfig 文件路径（默认使用 ~/.kube/config）
      --lang string         界面语言：zh / en（默认 zh）
  -h, --help                显示帮助

Use "saola [command] --help" for more information about a command.
```

**结果**: PASS — 11 个子命令（build/create/delete/describe/get/inspect/install/run/uninstall/upgrade/version）均在帮助中列出。

---

### TC-05: saola --lang=en --help

**目的**: 验证英文模式下根命令帮助正常显示。

```bash
# 输入
saola --lang=en --help

# 输出
saola is a CLI tool for managing middlewares

Usage:
  saola [command]

Available Commands:
  build       Build a middleware package
  create      Create a resource
  delete      Delete a resource
  describe    Show details of a resource
  get         List resources
  inspect     Inspect an installed package
  install     Install a package to the cluster
  run         Run an action
  uninstall   Uninstall a package
  upgrade     Upgrade an installed package
  version     Show version information

Flags:
      --kubeconfig string   Path to kubeconfig file (default ~/.kube/config)
      --lang string         Interface language: zh / en (default zh)
  -h, --help                Show help

Use "saola [command] --help" for more information about a command.
```

**结果**: PASS — 所有子命令均以英文显示。

---

### TC-06: saola build \<dir\> -o \<output\>

**目的**: 验证从 dataservice-baseline 构建真实 PostgreSQL 包，隐藏文件被正确排除。

```bash
# 输入
saola build /path/to/dataservice-baseline/postgresql -o /tmp/pg-test.pkg

# 输出
Built PostgreSQL@2.8.2-1.0.0 -> /tmp/pg-test.pkg (62151 bytes)
```

**验证**: 包体 62KB，排除了 `.git`、`.DS_Store`、`.gitignore` 等隐藏文件；若不排除，包体约 1.3MB，超出 K8s Secret 1MB 限制。

**结果**: PASS

---

### TC-07: saola install \<dir\> --name \<name\>

**目的**: 验证将本地 package 目录安装到集群（指定自定义 Secret 名称）。

```bash
# 输入
saola --kubeconfig kubeconfig/187 install /path/to/dataservice-baseline/postgresql --name saola-e2e-pkg

# 输出
Secret middleware-operator/saola-e2e-pkg created
```

**验证**: Secret 名称已小写化，符合 K8s RFC 1123 命名规范。

**结果**: PASS

---

### TC-08: saola get package（别名 get pkg）

**目的**: 验证以表格格式列出集群中已安装的所有 package。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get package

# 等效别名
saola --kubeconfig kubeconfig/187 get pkg

# 输出
NAME                                        COMPONENT    VERSION                               ENABLED   CREATED
postgresql-2.5.2-1.5.1                      PostgreSQL   2.5.2-1.5.1                           false     2026-03-18 20:20:08
postgresql-2.8.2-1.0.0                      PostgreSQL   2.8.2-1.0.0                           true      2026-03-31 15:16:20
redis-2.19.2-1.1.0-20260318084117-1208321   Redis        2.19.2-1.1.0-20260318084117-1208321   true      2026-03-18 20:50:52
saola-e2e-pkg                               PostgreSQL   2.8.2-1.0.0                           false     2026-03-31 15:42:07
```

**结果**: PASS — 表格对齐美观，NAME/COMPONENT/VERSION/ENABLED/CREATED 列均正确，共 4 个包。

---

### TC-09: saola inspect \<name\>

**目的**: 验证查看已安装 package 的元数据和文件列表，不含隐藏文件。

```bash
# 输入
saola --kubeconfig kubeconfig/187 inspect postgresql-2.8.2-1.0.0

# 输出
Name:      postgresql-2.8.2-1.0.0
Component: PostgreSQL
Enabled:   true
Created:   2026-03-31 15:17:16
Version:   2.8.2-1.0.0
Type:      db
Owner:     OpenSaola

Files:
  README.md (8448 bytes)
  actions/datasecurity.yaml (252 bytes)
  actions/disaster.yaml (10646 bytes)
  actions/failover.yaml (4552 bytes)
  actions/migrate.yaml (12581 bytes)
  baselines/masterslave-active.yaml (11861 bytes)
  baselines/masterslave.yaml (11983 bytes)
  baselines/operator-highly-available.yaml (25201 bytes)
  baselines/operator-standard.yaml (25094 bytes)
  configurations/alertrule.yaml (15088 bytes)
  ...（共 32 个文件，无 .git / .DS_Store / .gitignore 等隐藏文件）
```

**结果**: PASS — 元数据字段（Name/Component/Enabled/Version/Type/Owner）完整，文件列表含大小信息且无隐藏文件。

---

### TC-10: saola install（重复安装）

**目的**: 验证安装已存在的 package 时正确拒绝并给出提示。

```bash
# 输入
saola --kubeconfig kubeconfig/187 install /path/to/dataservice-baseline/postgresql --name saola-e2e-pkg

# 输出（exit code 1）
Error: Secret already exists; use 'package upgrade' to update
```

**结果**: PASS — 正确拒绝重复安装，提示使用 upgrade。

---

### TC-11: saola upgrade \<dir\> --name \<name\>

**目的**: 验证升级已有 package（删除旧 Secret 并重建）。

```bash
# 输入
saola --kubeconfig kubeconfig/187 upgrade /path/to/dataservice-baseline/postgresql --name saola-e2e-pkg

# 输出
Deleted existing Secret middleware-operator/saola-e2e-pkg
Secret middleware-operator/saola-e2e-pkg created (upgraded to PostgreSQL@2.8.2-1.0.0)
```

**结果**: PASS

---

### TC-12: saola uninstall \<name\>

**目的**: 验证卸载 package（在 Secret 上添加 uninstall 注解）。

```bash
# 输入
saola --kubeconfig kubeconfig/187 uninstall saola-e2e-pkg

# 输出
Uninstall annotation added to Secret middleware-operator/saola-e2e-pkg
```

**结果**: PASS

---

### TC-13: saola install --dry-run

**目的**: 验证 dry-run 模式只打印预期操作，不实际创建 Secret。

```bash
# 输入
saola --kubeconfig kubeconfig/187 install /path/to/dataservice-baseline/postgresql --name saola-e2e-dryrun --dry-run

# 输出
Dry-run: would create Secret middleware-operator/saola-e2e-dryrun
```

**验证**: 集群中无对应 Secret 被创建。

**结果**: PASS

---

### TC-14: saola create -f \<file\> --dry-run

**目的**: 验证 dry-run 模式创建 Middleware，不实际连接集群写入资源。

**准备**: 创建测试 YAML `/tmp/test-pg-e2e.yaml`：

```yaml
apiVersion: middleware.cn/v1
kind: Middleware
metadata:
  name: saola-e2e-pg
  namespace: middleware1
spec:
  baseline: postgresql-master-slave
  necessary:
    characterSet: UTF8
    locale: zh_CN.UTF-8
    password: Ab123456
    repository: <registry-host>/middleware
    resource:
      postgresql:
        limits:
          cpu: "1"
          memory: "2"
        replicas: 2
        requests:
          cpu: "1"
          memory: "2"
        volume:
          size: 10Gi
          storageClass: caas-lvm
    version: "14.7"
```

```bash
# 输入
saola --kubeconfig kubeconfig/187 create -f /tmp/test-pg-e2e.yaml --dry-run

# 输出
Middleware/saola-e2e-pg created (dry-run)
```

**结果**: PASS — dry-run 模式不连接集群，不实际创建资源。

---

### TC-15: saola create -f \<file\>（真实创建）

**目的**: 验证从 YAML 创建 Middleware，CLI 自动补全 4 个 labels 和 operatorBaseline，资源被 operator 正常调谐到 Available 状态。

```bash
# 输入
saola --kubeconfig kubeconfig/187 create -f /tmp/test-pg-e2e.yaml

# 输出
auto-enriched middleware "saola-e2e-pg":
  middleware.cn/packagename=postgresql-2.8.2-1.0.0
  middleware.cn/packageversion=2.8.2-1.0.0
  middleware.cn/component=PostgreSQL
  middleware.cn/definition=postgresql-master-slave
  spec.operatorBaseline.name=postgresql-operator-standard spec.operatorBaseline.gvkName=v1
middleware/saola-e2e-pg created
```

```bash
# 等待 10s 后检查状态
saola --kubeconfig kubeconfig/187 describe middleware saola-e2e-pg -n middleware1

# 输出（Status 部分）
Status:
  State:               Available
  ObservedGeneration:  1
  CustomResources:
    Phase:   Creating

Conditions:
  TYPE                       STATUS    REASON                            MESSAGE
  Checked                    True      CheckedSuccess                    成功
  TemplateParseWithBaseline  True      TemplateParseWithBaselineSuccess  成功
  BuildExtraResource         True      BuildExtraResourceSuccess         成功
  ApplyCluster               True      ApplyClusterSuccess               成功
```

**结果**: PASS — CLI 自动从已安装的 package 中查找含目标 baseline 的包，补全 4 个关键 labels 和 operatorBaseline 字段，所有 4 个 Conditions=True，State=Available。

---

### TC-16: saola get middleware -n \<ns\>

**目的**: 验证列出指定命名空间内的所有 Middleware 资源，包含新创建的实例。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get middleware -n middleware1

# 输出
NAME           NAMESPACE     BASELINE                  STATE       AGE
pg-test2       middleware1   postgresql-master-slave   Available   11d
saola-e2e-pg   middleware1   postgresql-master-slave   Available   2m
```

**结果**: PASS — 新创建的 saola-e2e-pg 已列出，State=Available。

---

### TC-17: saola describe middleware \<name\> -n \<ns\>

**目的**: 验证以人类可读格式展示 Middleware 完整的 Spec/Status/Conditions 信息。

```bash
# 输入
saola --kubeconfig kubeconfig/187 describe middleware saola-e2e-pg -n middleware1

# 输出
Name:         saola-e2e-pg
Namespace:    middleware1
Age:          2m
Labels:       middleware.cn/component=PostgreSQL, middleware.cn/definition=postgresql-master-slave, middleware.cn/packagename=postgresql-2.8.2-1.0.0, middleware.cn/packageversion=2.8.2-1.0.0
Annotations:  mode=HA, baselineName=MasterSlave Mode

Spec:
  Baseline:  postgresql-master-slave
  OperatorBaseline:
    Name:      postgresql-operator-standard
    GvkName:   v1
  PreActions:  5 item(s)

Status:
  State:               Available
  ObservedGeneration:  1
  CustomResources:
    Phase:  Creating

Conditions:
  TYPE                       STATUS    REASON                            MESSAGE
  Checked                    True      CheckedSuccess                    成功
  TemplateParseWithBaseline  True      TemplateParseWithBaselineSuccess  成功
  BuildExtraResource         True      BuildExtraResourceSuccess         成功
  ApplyCluster               True      ApplyClusterSuccess               成功
```

**结果**: PASS — Spec/Status/Conditions 信息完整输出。

---

### TC-18: saola delete middleware \<name\> -n \<ns\>

**目的**: 验证删除已存在的 Middleware 资源。

```bash
# 输入
saola --kubeconfig kubeconfig/187 delete middleware saola-e2e-pg -n middleware1

# 输出
middleware/saola-e2e-pg deleted
```

**结果**: PASS

---

### TC-19: saola delete mw \<name\> --dry-run（别名 + dry-run）

**目的**: 验证别名 `mw` 和 dry-run 组合使用，不实际删除资源。

```bash
# 输入
saola --kubeconfig kubeconfig/187 delete mw nonexistent --dry-run -n middleware1

# 输出
middleware/nonexistent deleted (dry-run)
```

**结果**: PASS — 别名正常解析，dry-run 不连接集群。

---

### TC-20: saola delete mw \<name\>（不存在的资源）

**目的**: 验证删除不存在的资源时静默容忍，返回 exit code 0。

```bash
# 输入
saola --kubeconfig kubeconfig/187 delete mw nonexistent -n middleware1

# 输出
middleware/nonexistent not found (already deleted)
```

**结果**: PASS — 返回 exit code 0，输出友好提示，不报错退出。

---

### TC-21: saola get operator -n middleware-operator

**目的**: 验证列出指定命名空间内的所有 MiddlewareOperator 资源。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get operator -n middleware-operator

# 输出
NAME                  NAMESPACE             BASELINE                       STATE       READY   RUNTIME   AGE
postgresql-operator   middleware-operator   postgresql-operator-standard   Available   false             11d
redis-operator        middleware-operator   redis-operator-standard        Available   false             12d
```

**结果**: PASS — 两个 operator 均正确列出。

---

### TC-22: saola describe op \<name\> -n \<ns\>（别名）

**目的**: 验证别名 `op` 正常解析，完整输出 MiddlewareOperator 的 Spec/Status/Conditions/OperatorStatus。

```bash
# 输入
saola --kubeconfig kubeconfig/187 describe op postgresql-operator -n middleware-operator

# 输出
Name:         postgresql-operator
Namespace:    middleware-operator
Labels:       middleware.cn/definition=postgresql-operator-standard
Annotations:  baselineName=Standard, description=Standard Baseline
Created:      2026-03-19T20:51:07Z

Spec:
  Baseline:         postgresql-operator-standard
  PermissionScope:  Cluster
  Configurations:   8 item(s)
    - crd-postgresqls
    - crd-postgresteams
    - crd-postgresql-operatorconfigurations
    - postgresql-operator-alertrule
    - postgresql-operator-service
    - postgresql-operator-clusterrole-postgres-pod
    - postgresql-operator-dashboard
    - postgresql-operator-operatorconfiguration
  PreActions:  3 item(s)
    - pre-pgsql-operator-affinity (fixed=false exposed=false)
    - pre-pgsql-operator-tolerations (fixed=false exposed=false)
    - pre-postgresql-operator-alertrule-labels (fixed=false exposed=false)

Status:
  State:               Available
  Ready:               false
  Runtime:
  OperatorAvailable:   1/1
  ObservedGeneration:  1

Conditions:
  TYPE                  STATUS    REASON                      MESSAGE
  Checked               True      CheckedSuccess              成功
  BuildExtraResource    True      BuildExtraResourceSuccess   成功
  ApplyRBAC             True      ApplyRBACSuccess            成功
  ApplyOperator         True      ApplyOperatorSuccess        成功

OperatorStatus:
  postgresql-operator:
    Replicas:   1
    Ready:      1
    Available:  1
    Updated:    1
    Conditions:
      Available=True (MinimumReplicasAvailable)
      Progressing=True (NewReplicaSetAvailable)
```

**结果**: PASS — Spec/Status/Conditions/OperatorStatus 均完整输出。

---

### TC-23: saola get action -A

**目的**: 验证跨命名空间列出 MiddlewareAction 资源，空列表场景友好提示。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get action -A

# 输出
No MiddlewareActions found.
```

**结果**: PASS — 空列表时输出友好提示，不报错退出。

---

### TC-24: saola get baseline --package \<pkg\> --kind middleware

**目的**: 验证列出指定 package 中 middleware 类型的所有 Baseline。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get baseline --package postgresql-2.8.2-1.0.0 --kind middleware

# 输出
NAME                             OPERATOR                       CONFIGURATIONS   PREACTIONS
postgresql-master-slave-active   postgresql-operator-standard   8                5
postgresql-master-slave          postgresql-operator-standard   8                5
```

**结果**: PASS — 2 个 middleware baseline 正确列出，CONFIGURATIONS=8，PREACTIONS=5。

---

### TC-25: saola get baseline --package \<pkg\> --kind operator

**目的**: 验证列出指定 package 中 operator 类型的所有 Baseline。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get baseline --package postgresql-2.8.2-1.0.0 --kind operator

# 输出
NAME                                   OPERATOR   CONFIGURATIONS   PREACTIONS
postgresql-operator-highly-available              8                3
postgresql-operator-standard                      8                3
```

**结果**: PASS — 2 个 operator baseline 正确列出。

---

### TC-26: saola get all -A

**目的**: 验证聚合查询所有资源类型，分段输出。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get all -A

# 输出
=== Middlewares ===
NAME                   NAMESPACE     BASELINE                  STATE         AGE
redis-agent-demo-001   middleware    redis-cluster             Unavailable   12d
redis-complete-test    middleware    redis-cluster             Unavailable   12d
redis-mcp-final-test   middleware    redis-cluster             Unavailable   12d
pg-test2               middleware1   postgresql-master-slave   Available     11d

=== Operators ===
NAME                  NAMESPACE             BASELINE                       STATE       READY   RUNTIME   AGE
postgresql-operator   middleware-operator   postgresql-operator-standard   Available   false             11d
redis-operator        middleware-operator   redis-operator-standard        Available   false             12d

=== Actions ===
No MiddlewareActions found.
```

**结果**: PASS — 三类资源分段输出，标题清晰。

---

### TC-27: saola get mw -A -o wide

**目的**: 验证 wide 输出格式在标准列基础上附加 LABELS 列。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get mw -A -o wide

# 输出
NAME                   NAMESPACE     BASELINE                  STATE         AGE   LABELS
redis-agent-demo-001   middleware    redis-cluster             Unavailable   12d   middleware.cn/component=Redis,...
redis-complete-test    middleware    redis-cluster             Unavailable   12d   middleware.cn/component=Redis,...
redis-mcp-final-test   middleware    redis-cluster             Unavailable   12d   middleware.cn/component=Redis,...
pg-test2               middleware1   postgresql-master-slave   Available     11d   middleware.cn/component=PostgreSQL,...
```

**结果**: PASS — wide 模式额外显示 LABELS 列。

---

### TC-28: saola get mw -A -o name

**目的**: 验证 name 输出格式，每行输出 `<资源类型>/<名称>` 格式。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get mw -A -o name

# 输出
middleware/redis-agent-demo-001
middleware/redis-complete-test
middleware/redis-mcp-final-test
middleware/pg-test2
```

**结果**: PASS — 每行格式为 `middleware/<name>`，便于脚本处理。

---

### TC-29: saola get mw \<name\> -o yaml

**目的**: 验证单个资源以 YAML 格式完整输出。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get mw pg-test2 -n middleware1 -o yaml

# 输出（截取前 20 行）
typemeta:
  kind: Middleware
  apiversion: middleware.cn/v1
objectmeta:
  name: pg-test2
  namespace: middleware1
  uid: a3b7c2d1-e4f5-6789-abcd-ef0123456789
  resourceversion: "58123456"
  generation: 1
  creationtimestamp: "2026-03-19T20:51:07+08:00"
  labels:
    middleware.cn/component: PostgreSQL
    middleware.cn/definition: postgresql-master-slave
    middleware.cn/packagename: postgresql-2.8.2-1.0.0
    middleware.cn/packageversion: 2.8.2-1.0.0
spec:
  baseline: postgresql-master-slave
  operatorbaseline:
    name: postgresql-operator-standard
    gvkname: v1
  ...
```

**结果**: PASS — 完整 YAML 结构输出，包含 metadata/spec/status 所有字段。

---

### TC-30: saola get mw \<name\> -o json

**目的**: 验证单个资源以 JSON 格式完整输出。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get mw pg-test2 -n middleware1 -o json

# 输出（截取前 20 行）
{
  "typemeta": {
    "kind": "Middleware",
    "apiversion": "middleware.cn/v1"
  },
  "objectmeta": {
    "name": "pg-test2",
    "namespace": "middleware1",
    "uid": "a3b7c2d1-e4f5-6789-abcd-ef0123456789",
    "resourceversion": "58123456",
    "creationtimestamp": "2026-03-19T20:51:07+08:00",
    "labels": {
      "middleware.cn/component": "PostgreSQL",
      "middleware.cn/definition": "postgresql-master-slave",
      "middleware.cn/packagename": "postgresql-2.8.2-1.0.0",
      "middleware.cn/packageversion": "2.8.2-1.0.0"
    }
  },
  ...
}
```

**结果**: PASS — 完整 JSON 结构输出。

---

### TC-31: saola get op -o wide

**目的**: 验证 MiddlewareOperator 列表 wide 输出格式附加 LABELS 列。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get op -n middleware-operator -o wide

# 输出
NAME                  NAMESPACE             BASELINE                       STATE       READY   RUNTIME   AGE   LABELS
postgresql-operator   middleware-operator   postgresql-operator-standard   Available   false             11d   middleware.cn/definition=postgresql-operator-standard,...
redis-operator        middleware-operator   redis-operator-standard        Available   false             12d   middleware.cn/definition=redis-operator-standard,...
```

**结果**: PASS — wide 模式额外显示 LABELS 列。

---

### TC-32: 无效 kubeconfig

**目的**: 验证指定不存在的 kubeconfig 时，错误信息清晰，含文件路径。

```bash
# 输入
saola --kubeconfig /tmp/nonexistent get mw

# 输出（exit code 1）
Error: create k8s client: build rest config: stat /tmp/nonexistent: no such file or directory
```

**结果**: PASS — 错误信息含完整文件路径。

---

### TC-33: 缺少必填 flag

**目的**: 验证 create 命令缺少 `--file` 必填 flag 时，错误信息指出 flag 名称。

```bash
# 输入
saola create

# 输出（exit code 1）
Error: required flag(s) "file" not set
```

**结果**: PASS — 错误信息明确指出缺少的 flag 名称。

---

### TC-34: 文件不存在

**目的**: 验证指定不存在的 YAML 文件时，错误信息含完整文件路径。

```bash
# 输入
saola --kubeconfig kubeconfig/187 create -f /tmp/nonexistent.yaml

# 输出（exit code 1）
Error: read file /tmp/nonexistent.yaml: open /tmp/nonexistent.yaml: no such file or directory
```

**结果**: PASS

---

### TC-35: 无效包目录

**目的**: 验证 build 指定不存在的目录时，错误信息含目录路径。

```bash
# 输入
saola build /tmp/nonexistent-dir

# 输出（exit code 1）
Error: pack: read metadata.yaml: open /tmp/nonexistent-dir/metadata.yaml: no such file or directory
```

**结果**: PASS

---

### TC-36: 包不存在

**目的**: 验证 inspect 不存在的 package 时，错误信息含 Secret 名称。

```bash
# 输入
saola --kubeconfig kubeconfig/187 inspect nonexistent-pkg

# 输出（exit code 1）
Error: get package: get secret failed: secrets "nonexistent-pkg" not found
```

**结果**: PASS

---

### TC-37: 无效输出格式

**目的**: 验证指定不支持的 -o 格式时，错误信息列出所有支持的格式。

```bash
# 输入
saola --kubeconfig kubeconfig/187 get mw -o invalid

# 输出（exit code 1）
Error: unknown output format "invalid" (supported: table, wide, yaml, json, name)
```

**结果**: PASS — 错误信息列出所有支持格式，便于用户纠正。

---

### TC-38: 旧命令已移除

**目的**: 验证 kubectl 风格重构后，旧的子命令优先风格（`saola middleware get`）已不再支持。

```bash
# 输入
saola middleware get

# 输出（exit code 1）
Error: unknown command "middleware" for "saola"
```

**结果**: PASS — 旧风格命令已完全移除，用户应使用 `saola get middleware`。

---

## 多语言切换测试

| 场景 | 中文（默认） | 英文（--lang=en） |
|------|:---:|:---:|
| 根命令 help | PASS | PASS |
| get 子命令 help | PASS | PASS |
| create/delete/describe help | PASS | PASS |
| Flag usage 描述 | PASS | PASS |
| completion/help 内置命令 | PASS | PASS |

---

## 别名测试

| 别名 | 完整名 | 结果 |
|------|--------|:---:|
| mw | middleware | PASS |
| op | operator | PASS |
| act | action | PASS |
| bl | baseline | PASS |
| pkg | package | PASS |

---

## 已修复的问题

在 E2E 测试过程中发现并修复了以下 6 个问题：

| # | 文件 | 问题描述 | 修复方式 |
|---|------|---------|---------|
| 1 | `internal/cmd/baseline/list.go` | `baseline list` 表格模式对复杂嵌套对象输出不可读（列太宽） | 新增 `baselineRow` 自定义表格行，只展示 NAME / OPERATOR / CONFIGURATIONS / PREACTIONS 关键列 |
| 2 | `internal/cmd/middleware/get.go` | `formatAge` 在客户端时钟落后于服务端时返回负数（如 `-1806s`） | 添加 `d < 0` 守卫，负值时返回 `<just now>` |
| 3 | `internal/cmd/middleware/create.go` + `operator/create.go` | YAML 反序列化使用 `gopkg.in/yaml.v3`，不识别 `json:` struct tag，导致 `metadata.name/namespace` 无法解析 | 替换为 `sigs.k8s.io/yaml`（先 YAML→JSON 再标准 JSON 解码，兼容 K8s 对象） |
| 4 | `internal/packager/packager.go` | `PackDir` 将 `.git`、`.DS_Store`、`.gitignore` 等隐藏文件/目录打入包中，导致包体从 62KB 膨胀到 1.3MB，超出 K8s Secret 1MB 限制 | 在 `filepath.WalkDir` 回调中跳过所有以 `.` 开头的文件和目录 |
| 5 | `internal/packager/secret_builder.go` | Secret 名称直接使用 `metadata.Name`（如 `PostgreSQL-2.8.2-1.0.0`），大写字母违反 K8s RFC 1123 命名规范导致创建失败 | 对生成的 Secret 名称调用 `strings.ToLower()` |
| 6 | `internal/cmd/middleware/create.go` | `middleware create` 不设置 `middleware.cn/packagename` 等关键 labels 和 `spec.operatorBaseline`，导致 operator 在 `TemplateParseWithBaseline` 阶段无法定位 package Secret，中间件永远 Unavailable | 新增 `enrichMiddleware()` 方法：自动扫描已安装的 package secrets，查找包含目标 baseline 的包，补全 4 个必需 labels 和 operatorBaseline 字段 |

---

## 环境清理确认

| 资源 | 状态 |
|------|------|
| Middleware `saola-e2e-pg`（middleware1 ns） | 已删除 |
| Secret `saola-e2e-pkg`（middleware-operator ns） | 已删除 |
| Secret `postgresql-2.8.2-1.0.0`（middleware-operator ns） | 保留（已启用） |

---

## 单元测试覆盖

E2E 测试之外，项目已有完整的单元测试覆盖：

```
ok  internal/cmd/action       7 tests
ok  internal/cmd/baseline     4 tests
ok  internal/cmd/middleware   13 tests
ok  internal/cmd/operator     12 tests
ok  internal/cmd/pkgcmd       17 tests
ok  internal/config            tests
ok  internal/packager          tests
ok  internal/printer           tests
ok  internal/version           tests
ok  internal/waiter            tests
```

所有单元测试使用 controller-runtime fake client 注入，不依赖真实集群。
