# SSA 迁移验证记录（2026-09-09）

结论：Nginx/Caddy 完整升级与运行测试通过；Nacos、Redis、MySQL 的 API 字段迁移验证通过，业务运行验收因镜像拉取失败尚未完成。本记录不等同于所有客户历史资源均满足迁移前提。

## 实际升级环境

- 独立 Kind：`opensaola-ssa-upgrade`，Kubernetes v1.34.3，单节点 Linux arm64，开启 RequestResponse 审计。
- 旧版：从 OpenSaola `v1.1.1` tag 原样导出并编译，commit `6db8d612b61cb2c704cc7be6c4d143ff7004f964`。
  二进制在容器中以 `/app/manager` 运行；未修改旧版源码，不是人工模拟 Create 替代真实控制器。
- 包：Nginx `1.29.3-1.0.0`、Caddy `2.11.0-1.0.0`，分别使用 `nginx-standard` / `caddy-standard`。
- CLI 执行包 validate、build、install、create。正式包文件没有修改。
- 每个实例单副本，requests 100m/128Mi，limits 200m/192Mi，无 PVC。
- 临时 MID 加入测试 env、emptyDir/volumeMount；临时 MCF 生成 ConfigMap 和 Service。

## 最终版本完整演练

| 步骤 | 结果 |
|---|---|
| 1.1.1 发布 `ssa-nginx-final`、`ssa-caddy-final` | 两实例成功创建并提供 HTTP 服务 |
| 添加外部 manager 字段，移除 MID/MCF 中的旧期望 | generation/observedGeneration 均为 2，但旧 env、volume/mount、ConfigMap key 仍残留 |
| CLI plan | 6 个新目标 ready：2 Deployment、2 ConfigMap、2 Service；先前试验已迁移对象跳过 |
| 停止旧控制器，CLI apply | 6 个目标迁移成功；每个写入均验证业务字段/status 不变，保留原对象备份 |
| 启动最终修复版 | 不再修改 MID spec/generation，启动 reconcile 完成正常 SSA |
| 字段检查 | `SSA_AUTH`、`ssa-legacy` volume/mount、ConfigMap `legacy` 删除 |
| 外部字段检查 | `HUMAN_KEEP`、共同拥有的 `SSA_SHARED`、ConfigMap `human` 保留；外部 managedFields 字段集与迁移前一致 |
| 运行验收 | 两个 Deployment 均 1/1 Ready，两个 Service HTTP 请求均成功 |
| CLI verify | 6 个迁移目标全部通过，包括 API 默认字段重新补回的情况 |

另以最终修复版直接创建 `ssa-nginx-fresh`、`ssa-caddy-fresh`：首次 root workflow 仅有
`opensaola/Apply`；不运行迁移命令，移除期望后同样成功删除旧字段、保留外部字段，HTTP 正常。

开发过程中先做过一组同类升级演练，metadata 迁移前后全部 Pod UID 一致。
真实演练发现并修复了审计 Patch 请求体解析，以及 API 默认化导致校验误判的问题。

## 自动化检查

- saola-cli：`go test ./...`、迁移包 race 检查、迁移命令/引擎 go vet 通过。
- CLI envtest：8 个真实 API Server 场景通过，涵盖子集迁移、同名 manager 后加/改值字段保留、外部共同所有权、
  证据缺失、错误 writer、旧计划/错误集群、schema 变化、所有权丢失、外部 owner、manager 复用，
  以及 CRD atomic map、关联列表和 status。
- 单元测试另覆盖包归档回退、计划误编辑、审计 JSON Patch 请求体与字段集拆分。
- OpenSaola：`go test ./...` 通过；启动兼容与暂停的 4 个 controller envtest 场景通过。

## 限制与证据

尚未证明 Nacos/Redis/MySQL 的业务运行安全、所有 CRD/operator、HA/故障切换或所有 Kubernetes 版本的兼容性。
跨作用域/外部 owner、缺少历史审计/schema 的客户资源仍会被阻断；正常 HTTP 检查不代表零中断 SLA。

原始日志、临时源码、审计、备份及验收 JSON 保存在本机 `/tmp/saola-ssa-upgrade-20260909/`；
审计和备份按敏感资料保管，不提交到仓库。最终验收断言见 `acceptance-final.json` 与 `fresh-acceptance.json`。

清理完成：临时 Kind 集群及本次构建的三个测试镜像已删除；默认 context 仍为 `kind-opensaola-formal`，
原有 OpenSaola deployment 保持 2/2 Ready。原始本机证据目录保留供复查。


## 主流中间件补充验证：字段通过，运行受阻

独立 Kind `opensaola-ssa-mainstream`（v1.34.3、Linux arm64）运行同一真实 1.1.1 二进制，
创建审计覆盖 `ssa-lab`；在创建阶段保存真实 OpenAPI 快照，随后保持目标 schema 不变。
包与夹具如下，包源码未修改：

| 中间件 | 包 / 运行版本 | 最小测试配置 | 包与运行结果 |
|---|---|---|---|
| Nacos | 1.6.1-1.0.0 / 2.5.1 | embedded-standalone，1 副本，1 CPU/1Gi，数据/日志 PVC 各 1Gi | validate/build/install/create 通过；Pod 镜像拉取失败 |
| Redis | 2.20.4-1.0.0 / 7.2.4 | sentinel，期望 2 Redis + 3 Sentinel；Redis request 100m/128Mi，limit 1 CPU/1Gi | validate/build 被两处模板校验错误阻断；install/create 成功；Operator 镜像拉取失败 |
| MySQL | 1.13.0-zjdl-1.0.0 / 8.0.39 | master-slave，期望 2 副本，每副本 request 100m/256Mi，limit 1 CPU/1Gi，PVC 1Gi | validate/build/install/create 通过；Operator 镜像拉取失败 |

MySQL 使用当前工作区的 zjdl 专线包，不能泛化为所有标准 MySQL 包。
Redis 校验错误分别为 `connection-credential.yaml` 渲染 YAML 解析失败及 `redis-config.yaml` 的
`semverCompare` 收到无效版本；本次没有修改校验器或包绕过 build，也不把 install 成功等同于校验通过。

验证步骤及实际断言：

1. Nacos `auth.enable=true` 创建后，将该字段改为 false，保留 password 输入键。
   1.1.1 已记录 `opensaola/Apply`，但 `manager/Update` 持有的旧认证 env 仍残留。
2. Redis/MySQL 各通过临时 MID Configuration 引用添加一个测试 ConfigMap；由旧控制器首次 Create。
   再从临时 MCF 删除 `data.legacy`，触发旧版 Apply，确认旧 key 仍残留。
   这些是附加字段夹具，不冒充原始包自带的条件 key。
3. 外部 manager 向 Nacos StatefulSet 写 `HUMAN_KEEP`，人工 manager 向三个真实包 ConfigMap
   及两个附加 ConfigMap 写 `data.human`。
4. 停止旧控制器后，plan 扫描 31 个资源：24 ready、7 blocked。24 个迁移成功；
   7 个集群级 CRD 因严格同命名空间 MID/MO owner 证明不足保持 blocked，apply 正确返回非零。
   实际 Nacos、RedisCluster、MysqlCluster 主 CR 均在此次 schema/历史证据下迁移成功；
   未将 Redis/MySQL 未启动的下游 Operator 子资源计入覆盖范围。
5. 对 31 个对象逐一比较 CLI 迁移前后：业务字段、status、外部 managedFields 全部不变；
   7 个 blocked CRD 的全部 managedFields 也不变。
6. 启动修复版，不再修改 MID spec，Nacos StatefulSet 的 `NACOS_AUTH_ENABLE`、
   `NACOS_AUTH_IDENTITY_KEY`、`NACOS_AUTH_IDENTITY_VALUE`、`NACOS_AUTH_TOKEN`、
   `NACOS_PASSWORD`（含 Secret 引用）全部消失，`HUMAN_KEEP` 保留。
   两个附加 ConfigMap 的 `legacy` 消失，`keep` 和人工 `human` 保留。
7. CLI verify：24 个 `ownership-migrated`，7 个仍 blocked，无其他失败。

运行阻塞证据：本机访问 `10.10.101.172:443` 超时；Kind 拉取 Nacos 及三个 Operator 镜像返回 EOF，
Pod 为 ErrImagePull/ImagePullBackOff。Redis/MySQL 下游工作负载尚未创建，Nacos Pod 未 Ready。
因此没有执行 Nacos 配置读写、Redis SET/GET、MySQL SQL 读写，不能据此声称迁移不影响运行中的中间件。
需要可达且包含对应镜像的仓库后补做“先 Ready/读写，再迁移，再 Ready/读写”的完整新实例演练。

证据目录：`/tmp/saola-ssa-mainstream-20260909/`，包含 `plan.json`、`apply.json`、`verify.json`、
`metadata-only-check.json`、资源前后快照、`image-events.json`、`runtime-after.json`。
原始审计/备份含敏感数据，仅保存在受限临时目录，不提交。
此补充环境保留以便镜像可用后续测；原默认集群未修改，其 OpenSaola 仍为 2/2 Ready。
