# Plugin Execution System

Plugin Execution System 是一个用 Go 实现的插件化任务执行平台。它的核心目标是：主程序不依赖具体插件实现，只负责插件发现、插件管理、任务调度、运行隔离、结果聚合、事件记录、审计和可验证交付。

本项目不是单文件 demo。它按企业后端分层组织代码：handler 只处理 HTTP，service 负责业务规则，repository 负责数据访问，worker 负责异步执行，runtime 负责插件运行隔离，response 负责统一响应，middleware 负责认证、日志、request_id、trace_id 和 recovery。

## 一句话定位

用户选择插件并提交输入数据，系统创建执行任务，worker 异步调度插件运行，runtime 隔离执行插件，系统保存结果、事件、attempt 和审计记录，最终通过 API、CLI 或前端查看任务状态和结果。

## 已实现的核心闭环

当前发布版已经实现并验证以下链路：

1. 启动 Go HTTP 服务。
2. 扫描 `plugins/` 目录并读取插件 manifest。
3. 管理员 reload 插件。
4. 管理员启用插件。
5. 用户创建 execution。
6. 系统校验插件状态和幂等键。
7. worker 异步消费任务。
8. runtime 通过进程型插件协议执行插件。
9. 插件成功、失败、非法输出、超时等情况被隔离处理。
10. 保存 execution results。
11. 聚合 summary。
12. 写入 events、attempts、audit。
13. 前端/API/CLI 可以查询任务、结果、事件、attempt、审计和 metrics。
14. `make verify` 可以一键验证完整闭环。

## 为什么采用进程型插件协议

本项目不使用 Go 标准库 `plugin` 包，也不直接接入现成流程引擎。主程序通过 manifest 描述插件，通过 stdin/stdout JSON 协议调用插件。

这样做的原因：

- 主程序不 import 插件代码，插件和平台解耦。
- 插件可以用 Python、Go、Node.js 等语言实现。
- 插件崩溃不会直接拖垮主进程。
- runtime 可以统一控制 timeout、stdout/stderr 限制、输出清洗、权限和资源策略。
- 后续可以平滑扩展到 container、wasm、remote runner。

## 快速验证

推荐直接运行：

```bash
make verify
```

这个命令会先执行：

```bash
go test ./...
```

然后运行闭环验证脚本：

```bash
python3 scripts/verify_closed_loop.py
```

验证脚本会自动启动一个临时 dev server，并完整检查：

- `/livez`
- `/readyz`
- `/workerz`
- `/dependencyz`
- 插件 reload
- 插件 enable
- 成功任务执行
- 幂等重复提交
- 幂等冲突
- worker 执行
- results 查询
- summary 查询
- events 查询
- attempts 查询
- audit 查询
- PartialSuccess 场景
- 分页
- metrics

验证通过时会输出：

```text
VERIFICATION PASSED
```

## 本地启动

本地演示模式使用 local-json metadata store。它只用于 local/dev，不作为生产路线。

```bash
APP_MODE=dev \
METADATA_STORE=local-json \
ALLOW_LOCAL_JSON_STORE=true \
SERVER_ADDR=:8080 \
go run ./cmd/server
```

访问：

```text
http://127.0.0.1:8080
```

健康检查：

```bash
curl http://127.0.0.1:8080/livez
curl http://127.0.0.1:8080/readyz
curl http://127.0.0.1:8080/workerz
curl http://127.0.0.1:8080/dependencyz
```

## 默认账号

本地演示默认 token：

```text
admin-token  管理员
 demo-token  普通用户
```

请求示例：

```bash
curl -H 'Authorization: Bearer admin-token' http://127.0.0.1:8080/api/v1/plugins
```

## 示例插件

项目内置 4 个示例插件：

- `echo`：原样返回输入。
- `text_stats`：统计文本字符数、单词数、行数。
- `json_pick`：从 JSON 中提取指定字段。
- `error_demo`：故意失败，用于验证错误隔离和 PartialSuccess。

## 主要 API

所有新接口以 `/api/v1` 为主。

插件：

```text
POST /api/v1/plugins/reload
GET  /api/v1/plugins?page=1&page_size=20
GET  /api/v1/plugins/{id}
POST /api/v1/plugins/{id}/enable
POST /api/v1/plugins/{id}/disable
```

任务：

```text
POST /api/v1/executions
GET  /api/v1/executions?page=1&page_size=20
GET  /api/v1/executions/{id}
POST /api/v1/executions/{id}/cancel
GET  /api/v1/executions/{id}/results
GET  /api/v1/executions/{id}/summary
GET  /api/v1/executions/{id}/events
GET  /api/v1/executions/{id}/attempts
```

审计与观测：

```text
GET /api/v1/audit/logs?page=1&page_size=20
GET /api/v1/audit/executions/{id}
GET /metrics
GET /debug/vars
```

详细接口见 `API_SPEC.md` 和 `docs/openapi.json`。

## 状态机

插件状态：

```text
Discovered -> Loaded -> Enabled
Enabled -> Disabled
Disabled -> Enabled
Loaded/Enabled/Disabled -> Error
Loaded/Disabled/Error -> Removed
```

任务状态：

```text
Pending -> Queued -> Running -> Success
Pending -> Queued -> Running -> PartialSuccess
Pending -> Queued -> Running -> Failed
Pending/Queued/Running -> Canceled
Running -> Timeout
```

所有关键状态变化都应该产生 event record，便于排查和审计。

## 幂等设计

创建任务支持 `Idempotency-Key`。

规则：

- 同一用户、同一 scope、同一 key、同一插件集合、同一输入 hash，返回同一个 execution。
- 同一 key 但输入或插件不同，返回 `IDEMPOTENCY_CONFLICT`。
- `make verify` 已覆盖重复提交和冲突场景。

## 安全边界

当前发布版已经实现或预留：

- request_id / trace_id。
- 统一错误码。
- 结构化日志。
- command allowlist。
- 插件 stdout/stderr 限制。
- 输出清洗。
- checksum / signature / provenance 字段。
- process / container / wasm / remote runner contract。
- RBAC / ABAC / policy engine 边界。
- secret reference + runtime injection 边界。
- webhook HMAC 签名和 retry/DLQ baseline。
- tenant_id / project_id 模型。

需要注意：本地 dev mode 以易验证为主；生产路线应使用 Postgres/HA metadata store、durable queue、container runner、OIDC/SAML、KMS/Vault、object storage、完整 OTel 和 release signing。

## 目录结构

核心目录：

```text
cmd/server              HTTP server 入口
cmd/pesctl              CLI
internal/model          领域模型和状态常量
internal/repository     数据访问接口与 adapter
internal/service        业务逻辑、状态机、幂等、插件管理、runtime
internal/handler        HTTP handler
internal/router         路由注册
internal/worker         worker pool、durable worker、webhook retry scheduler
internal/queue          durable queue 抽象
internal/storage        metadata store 抽象与迁移
internal/policy         RBAC/ABAC/policy engine
internal/secrets        secret provider
internal/artifact       artifact store
internal/security       输出清洗和安全工具
plugins                 示例插件
web/static              前端演示页面
scripts                 验证和发布辅助脚本
migrations/postgres     生产 metadata schema
```

更详细结构见 `CURRENT_STRUCTURE.md`。

## 测试

常用命令：

```bash
go test ./...
go test -race ./...
make verify
```

测试覆盖重点：

- 插件状态机。
- execution 状态机。
- manifest 解析。
- runtime 成功、失败、超时、非法输出。
- 幂等重复提交和冲突。
- API flow。
- worker 执行。
- webhook scope 和 retry。
- policy、secret、artifact、安全清洗。
- 闭环验证脚本。

## 生产路线说明

本发布版保留 local-json 作为 dev/local mode，方便评审和本地闭环验证。正式生产路线不能依赖 local-json，也不能依赖内存队列作为真相源。

生产标准路线：

- Metadata Store：Postgres / HA metadata store。
- Queue：Postgres `FOR UPDATE SKIP LOCKED`、NATS JetStream、Kafka 或 Temporal 等 durable execution 后端。
- Runtime：container runner 优先，process runner 仅作为 dev/compat。
- Artifact：S3/MinIO 或对象存储。
- Identity：OIDC/SAML/mTLS。
- Policy：OPA/Rego 或等价策略服务。
- Secret：Vault/KMS。
- Observability：OpenTelemetry + Prometheus。
- Release：checksum、SBOM、signature、attestation。

## 提交建议

如果用于笔试或项目评审，建议主打：

- 插件与主程序解耦。
- 插件发现、管理、启用/禁用。
- execution 异步执行闭环。
- runtime 错误隔离。
- results / summary / events / attempts / audit 可查询。
- `make verify` 一键证明闭环可运行。

不要把重点放在还未完全落地的企业扩展概念上。
