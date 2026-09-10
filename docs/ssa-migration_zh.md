# OpenSaola 1.1.1 存量 SSA 迁移

`saola migrate ssa` 把历史 Create/Update workflow 中有依据的字段迁移到 `opensaola/Apply`。
CLI 只改 metadata，不渲染或修改业务 spec/data，不触碰 status，不删除资源。后续由修复版 OpenSaola
正常 Apply 当前期望，删除已省略的字段。这个删除行为是修复目标，不需要另一轮字段删除审批。

## 命令

```sh
# 只读扫描；不提供历史资料时，输出阻断原因，不猜测字段来源。
saola --context customer -n application migrate ssa plan -o scan.json

# 使用创建期资料生成可执行计划；可用 -A 替代 -n 扫描所有 MID/MO。
saola --context customer -n application migrate ssa plan \
  --audit-log historical-audit.jsonl \
  --writer system:serviceaccount:middleware-operator:opensaola \
  --historical-openapi historical-openapi-v2.json \
  -o migration-plan.json

# 在维护窗口控制 OpenSaola 的写入，按已审阅计划迁移。
# 全部旧控制器副本应停止，或采用已验证的等效写入隔离方式。
saola --context customer migrate ssa apply \
  --plan migration-plan.json --backup-dir ./ssa-backups

# 启动修复版后再次校验；同时进行中间件功能验收。
saola --context customer migrate ssa verify --plan migration-plan.json
```

命令继承 `--kubeconfig`、`--context`、`--namespace`、`--pkg-namespace`；支持 `--timeout`，默认两分钟。
JSON 结果写 stdout，错误写 stderr。计划与备份使用独占创建和 0600 权限，备份目录必须为 0700。
计划不含完整业务对象，但字段路径中的关联列表键仍可能敏感；原始备份可能含 Secret 内容，应按敏感资料保管。

`apply` 对 ready 条目逐个执行，blocked/failed 条目保留原因并返回非零退出码，不能把部分成功视为全部成功。
发现权限或 discovery/list 错误的计划不可执行，防止把不完整扫描当成完整覆盖。
`verify` 证明字段归属迁移，不替代 Ready、Service 和实际协议检查。

## 如何分辨字段

必须分别确认资源身份和字段来源：

- MID 主资源：当前 MID UID、controller ownerReference、namespace/name、Baseline 声明 GVK 必须匹配。
- MCF：严格 owner UID、来源注解及当前 owner/Baseline 引用匹配。MCF 同名重建仍按逻辑名称识别。
- 同名 Baseline/MCF 被其他包版本覆盖时，按 owner 的包名从 `--pkg-namespace` 中的包 Secret 归档回退查找。
- 字段来源：读取成功的 RequestResponse 创建审计，校验真实 writer username、资源 UID、GVK/name/namespace。
  manager 名称与时间戳本身不是来源证明；发现同一 manager 被另一个审计主体复用时阻断。
- 对历史和当前 OpenAPI v2 的目标 GVK 及引用 schema 取摘要，要求一致。不跨版本盲目合并。

字段集合按 structured-merge-diff/schema 处理：

```text
H = 审计创建对象的字段 − 准入改写字段 − 创建后值发生变化的字段
L = 当前对应 root Update 的字段
T = L ∩ H

旧 Update 保留 L − T
opensaola/Apply 获得 原有字段 ∪ T
其他 manager 原样保留
```

共同拥有的字段在计划 `shared` 中列出，正常 SSA 省略时可能因其他 owner 而保留。
同名 manager 后来添加、但创建记录中没有的字段不迁移；变化值保守保留，即使变化可能来自合法 OpenSaola 写入。
Atomic 对象/列表按完整原子字段处理，不能拆成普通 JSON 路径迁移。status 和平台迁移标记不进入 T。

Kubernetes 审计 `requestObject` 是服务端解码后的对象，可能已包含默认值，不等于原始 HTTP 请求字节。
归属于可信创建 workflow 的默认字段也可能进入 T；后续 SSA 省略后 API Server 可以补回默认值且不保留 Apply ownership。
控制器通过 `middleware.cn/ssa-cli-converged` 记录迁移标识与实际 Apply 字段摘要，CLI 据此识别正常默认化。
如果业务字段仍存在、目标所有权丢失且没有匹配的收敛回执，校验失败，不能仅凭旧 Update 已不存在就报成功。

## 历史资料的边界

1.1.1 没有持久化原始渲染清单或历史 schema。本命令不能自动补造这些资料。
历史审计与 schema 文件必须来自可信归档，并覆盖相关写入历史；CLI 不为外部文件真实性提供签名认证。
`--writer` 指定的是已经核验的 OpenSaola ServiceAccount，不是通过 manager 名字推断的身份。
当前 schema 快照不能改名充当历史文件，当前模板也不能证明已经省略的旧字段。

当前支持有完整历史审计/schema 的严格同命名空间 MID/MO 资源。
跨命名空间、集群级 MCF、外部 controller augmentation、动态 PreAction 才能推导的引用、缺少历史资料、
跨 API version 或 schema 变化均失败关闭。不会提供 `--force-all-manager-fields` 绕过入口。
这意味着并非所有客户旧资源都可以直接迁移；缺资料的条目必须先解决来源证明，不能声称零配置覆盖。

## 并发、重试与升级

1. 只读 plan，核验目标集群与资源清单。不要手改计划，计划摘要防误编辑，不是签名。
2. 在维护窗口停止旧 OpenSaola 写入；不要同时改包、期望或 CRD/schema。业务 Pod 可以继续运行。
3. apply 重新读取 live/owner/schema，用 UID/RV test 进行 JSON Patch。先 dry-run，再保存原对象，然后实际写入。
4. 写入响应检查 spec/data 等业务字段、status、其他 managedFields 不变。对象变化时重试；依据已变化则要求新计划。
5. 字段转移与 `middleware.cn/ssa-cli-migration` 回执在同一次 Patch 完成；进程断开后用同一计划重试。
6. 启动修复版。控制器在 observedGeneration 提前返回之前处理带 CLI 回执的资源，不需要改 MID/MO spec。
7. 检查字段收敛、外部字段保留、rollout 和实际协议。正常删除 env/volume 可引起滚动更新。

UID/RV 条件是单资源保护，不是整个集群的事务；owner/schema 与资源无法用一个 Kubernetes Patch 原子锁定。
因此仍要求维护窗口保持配置/schema 稳定。发生部分失败时保留备份、回执和结果，不自动删除重建工作负载。

镜像回滚不会自动还原已删除字段；不得把完整旧 managedFields 无条件覆盖回去。
备份用于分析和制定有条件恢复方案，不提供破坏外部 manager 的一键恢复。
