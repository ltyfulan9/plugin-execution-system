# 插件执行系统发布版功能与测试说明

## 项目定位

这是一个 Go 实现的插件化任务执行系统。主程序不依赖具体插件实现，只负责插件发现、插件状态管理、任务创建、异步执行、运行隔离、结果聚合、事件记录、attempt 记录、审计和可观测性。

项目重点不是堆企业名词，而是做到一条可运行、可验证的完整闭环：

用户选择插件并提交输入 -> 系统创建 execution -> worker 异步执行 -> runtime 调用插件 -> 保存结果 -> 聚合状态 -> 写入 events/attempts/audit -> API/前端查询结果。

## 已实现功能

### 1. 插件发现与管理

已实现：

- 扫描 `plugins/` 目录。
- 解析 `manifest.json`。
- 支持 `plugin.exec/v1` manifest。
- 兼容 legacy flat manifest。
- 插件 reload。
- 插件列表查询。
- 插件详情查询。
- 插件启用。
- 插件禁用。
- 插件错误状态。

内置示例插件：

- `echo`：原样返回输入。
- `text_stats`：统计文本字符数、单词数、行数。
- `json_pick`：从 JSON 中提取指定字段。
- `error_demo`：故意返回失败，用于验证错误隔离和 PartialSuccess。

### 2. 插件执行协议

已实现进程型插件协议：

- 主程序通过 stdin 向插件写入 JSON。
- 插件通过 stdout 返回 JSON。
- stderr 作为诊断信息，会被截断和清洗。
- 插件成功、失败、非法输出、超时、运行错误都会映射为统一结果状态。

结果状态包括：

- `Success`
- `Failed`
- `Timeout`
- `InvalidOutput`
- `RuntimeError`

### 3. Execution 任务闭环

已实现：

- 创建 execution。
- `Idempotency-Key` 幂等控制。
- 相同 key + 相同输入返回同一个 execution。
- 相同 key + 不同输入返回 `IDEMPOTENCY_CONFLICT`。
- worker 异步执行任务。
- runtime 执行一个或多个插件。
- 单插件失败不影响主服务。
- 多插件部分成功时聚合为 `PartialSuccess`。
- 查询 execution 当前状态。
- 查询 results。
- 查询 summary。
- 查询 events。
- 查询 attempts。
- 查询 audit。

任务状态包括：

- `Pending`
- `Queued`
- `Running`
- `Success`
- `PartialSuccess`
- `Failed`
- `Timeout`
- `Canceled`

### 4. 分层架构

项目不是单文件玩具项目。当前结构按企业后端范式拆分：

- `cmd/server`：启动入口。
- `internal/model`：结构体、状态、枚举。
- `internal/repository`：数据访问接口和 adapter。
- `internal/service`：业务规则、状态机、幂等、runtime 调用。
- `internal/handler`：HTTP 参数解析和响应。
- `internal/router`：路由注册。
- `internal/worker`：异步任务执行。
- `internal/queue`：durable queue 抽象。
- `internal/storage`：metadata store 抽象。
- `internal/response`：统一 JSON 响应和错误码。
- `internal/middleware`：认证、日志、request_id、trace_id、recovery。
- `internal/security`：输出清洗、安全工具。
- `internal/policy`：RBAC/ABAC/policy engine 边界。
- `internal/secrets`：secret reference 边界。
- `internal/artifact`：artifact store 边界。

### 5. API 能力

新 API 以 `/api/v1` 为主。

插件接口：

```text
POST /api/v1/plugins/reload
GET  /api/v1/plugins?page=1&page_size=20
GET  /api/v1/plugins/{id}
POST /api/v1/plugins/{id}/enable
POST /api/v1/plugins/{id}/disable
```

任务接口：

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

健康检查：

```text
GET /livez
GET /readyz
GET /workerz
GET /dependencyz
```

### 6. 可观测性

已实现：

- `/metrics` Prometheus text baseline。
- `/debug/vars` expvar。
- request_id。
- trace_id。
- 结构化日志 baseline。
- execution events。
- execution attempts。
- audit logs。

### 7. 安全与企业扩展边界

已实现或预留：

- command allowlist。
- 路径逃逸检查。
- timeout 限制。
- stdout/stderr 大小限制。
- 输出清洗。
- checksum / signature / provenance 字段。
- process / container / wasm / remote runner contract。
- tenant_id / project_id 模型。
- RBAC / ABAC / policy engine 边界。
- secret reference + runtime injection 边界。
- artifact store 边界。
- webhook HMAC、retry、DLQ baseline。
- Postgres schema 和 HA metadata store 方向。

需要说明：当前用于闭环验证的是 local/dev mode。正式生产路线应使用 Postgres/HA metadata store、durable queue、container runner、OIDC/SAML、Vault/KMS、object storage、OpenTelemetry 和 release signing。

## 如何测试

### 推荐测试命令

```bash
make verify
```

这个命令会执行：

```bash
go test ./...
python3 scripts/verify_closed_loop.py
```

### 闭环验证内容

`make verify` 会自动验证：

1. 启动临时 server。
2. 检查 `/livez`。
3. 检查 `/readyz`。
4. 检查 `/workerz`。
5. 检查 `/dependencyz`。
6. reload 插件。
7. enable `echo / text_stats / error_demo`。
8. 创建成功任务。
9. 验证相同 `Idempotency-Key` 返回同一个 execution。
10. 验证冲突 `Idempotency-Key` 返回 `IDEMPOTENCY_CONFLICT`。
11. 等待 worker 执行完成。
12. 查询 summary。
13. 查询 results。
14. 查询 events。
15. 查询 attempts。
16. 查询 audit。
17. 创建 PartialSuccess 任务。
18. 验证分页。
19. 验证 metrics。

通过时输出：

```text
VERIFICATION PASSED
```

### 已复测结果

当前发布包已实际运行：

```bash
go test ./...
make verify
```

测试通过，闭环脚本输出：

```text
VERIFICATION PASSED
```

## 评审时建议怎么介绍

建议这样介绍：

> 我实现的是一个 Go 插件化任务执行系统。主程序不依赖插件实现，通过 manifest 和 stdin/stdout JSON 协议调用插件。系统支持插件发现、插件启用禁用、任务创建、worker 异步执行、runtime 隔离、结果聚合、事件、attempt、审计和统一 API。项目按 handler、service、repository、worker、runtime、model 分层，避免写成一个 main.go 或 service.go。可以通过 make verify 一键验证完整业务闭环。

不要重点宣传还未完全生产落地的能力。企业能力可以作为扩展方向说明。

## 最终结论

当前发布版已经满足插件化执行系统题目的核心要求，并且具备明显的工程化差异：

- 不是硬编码插件函数。
- 不是同步 demo。
- 不是单文件玩具项目。
- 有完整任务状态流转。
- 有错误隔离。
- 有结果聚合。
- 有事件、attempt、审计。
- 有统一响应和错误码。
- 有一键闭环验证。
