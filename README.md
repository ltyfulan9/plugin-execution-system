# Plugin Execution System

Plugin Execution System 是一个用 Go 实现的插件化任务执行平台。它的核心目标是：主程序不依赖具体插件实现，只负责插件发现、插件管理、任务调度、运行隔离、结果聚合、事件记录、审计和可验证交付。

本项目不是单文件 demo。它按企业后端分层组织代码：handler 只处理 HTTP，service 负责业务规则，repository 负责数据访问，worker 负责异步执行，runtime 负责插件运行隔离，response 负责统一响应，middleware 负责认证、日志、request_id、trace_id 和 recovery。

## 一句话定位

用户选择插件并提交输入数据，系统创建执行任务，worker 异步调度插件运行，runtime 隔离执行插件，系统保存结果、事件、attempt 和审计记录，最终通过 API、CLI 或前端查看任务状态和结果。

## 整体架构设计说明

本项目采用分层后端架构，不把所有逻辑堆在一个入口文件里，而是把 HTTP 接入、业务规则、数据访问、任务执行、插件运行和审计观测拆成相互独立的模块。

整体架构可以理解为：

用户 / CLI / 前端
→ HTTP API
→ Handler 层
→ Service 层
→ Repository 层
→ Worker 执行层
→ Runtime 插件运行层
→ 插件进程
→ 结果、事件、attempt、audit、metrics 持久化与查询

各层职责如下：

cmd/server：服务启动入口，负责加载配置、初始化依赖、启动 HTTP server 和 worker。
internal/router：统一注册路由，避免 handler 分散挂载。
internal/handler：处理 HTTP 请求、参数解析、鉴权上下文、响应封装，不写复杂业务逻辑。
internal/service：承载核心业务规则，包括插件管理、execution 创建、状态机流转、幂等校验、结果聚合和审计记录。
internal/repository：封装数据访问接口，屏蔽底层 metadata store 的具体实现。
internal/worker：异步消费 execution 任务，控制任务执行生命周期。
internal/runtime：负责真正调用插件，并处理 timeout、stdout/stderr 限制、异常隔离、非法输出等运行时问题。
internal/model：定义核心领域模型，例如 Plugin、Execution、Result、Event、Attempt、AuditLog。
internal/middleware：负责 request_id、trace_id、日志、鉴权、recovery 等通用能力。
plugins/：存放示例插件，每个插件通过 manifest 描述自身能力。
scripts/：提供闭环验证脚本，保证项目不是“能启动但不可验证”的半成品。

这种设计的核心目标是：主程序只负责任务编排和生命周期管理，不关心具体插件内部业务逻辑。

插件可以独立开发、独立发布、独立维护；主系统只通过统一协议调用插件，从而保证插件和平台之间解耦。

## 插件系统的核心设计思路

本项目的插件系统不是让主程序直接 import 插件代码，也不是使用 Go 标准库 plugin 包，而是采用：

manifest 描述插件 + stdin/stdout JSON 协议 + 独立进程执行

每个插件都需要在 plugins/ 目录下提供自己的 manifest。manifest 用来声明插件的基础信息和执行方式，例如：

插件 ID
插件名称
插件版本
插件执行命令
插件输入输出协议
timeout 配置
checksum / signature / provenance 等安全扩展字段

主程序启动或 reload 时会扫描插件目录，读取 manifest，把插件注册到系统中。插件本身不需要和主程序编译在一起。

执行插件时，runtime 会启动插件进程，把用户输入封装成 JSON 写入插件 stdin，然后从 stdout 读取插件返回的 JSON 结果。

这样设计有几个好处：

第一，主程序和插件彻底解耦。主程序只认识统一协议，不依赖某个具体插件的实现代码。

第二，插件语言不受限制。只要能读取 stdin、写出 stdout，就可以作为插件接入，因此后续可以支持 Go、Python、Node.js、Shell 等多种语言。

第三，插件异常不会直接拖垮主程序。插件 panic、进程退出、输出非法 JSON、执行超时，都只会影响当前 execution，不会导致 HTTP 服务崩溃。

第四，后续扩展空间更大。当前版本使用 process runner，后续可以平滑扩展到 container runner、wasm runner 或 remote runner。

插件系统围绕以下生命周期设计：

插件发现
→ manifest 解析
→ 插件注册
→ 插件启用 / 禁用
→ 用户选择插件创建 execution
→ worker 调度执行
→ runtime 调用插件
→ 保存结果
→ 记录事件和审计
→ 聚合 summary
→ 查询执行结果

插件状态机包括：

Discovered -> Loaded -> Enabled
Enabled -> Disabled
Disabled -> Enabled
Loaded/Enabled/Disabled -> Error
Loaded/Disabled/Error -> Removed

execution 状态机包括：

Pending -> Queued -> Running -> Success
Pending -> Queued -> Running -> PartialSuccess
Pending -> Queued -> Running -> Failed
Pending/Queued/Running -> Canceled
Running -> Timeout

这样可以保证插件和任务都有明确状态，不会出现状态混乱、执行结果不可追踪的问题。

## 关键实现与取舍(如下)

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

项目内置 6 个示例插件：

- `echo`：原样返回输入。
- `text_stats`：统计文本字符数、单词数、行数。
- `json_pick`：从 JSON 中提取指定字段。
- `data_quality`：检查 JSON 记录中的缺失字段、空值和重复行。
- `keyword_audit`：检查文本中的关键词覆盖率和出现频次。
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
