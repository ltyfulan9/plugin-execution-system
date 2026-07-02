# Test Plan

## 测试目标

测试目标不是只证明代码能编译，而是证明插件执行闭环真实可用：插件可以被发现、启用、执行、隔离、聚合结果，并且可以通过 API 查询完整执行轨迹。

## 必跑命令

```bash
go test ./...
make verify
```

建议在最终提交前额外运行：

```bash
go test -race ./...
```

## 单元测试重点

### 状态机

覆盖：

- 插件合法状态流转。
- 插件非法状态流转。
- 禁用插件不可执行。
- execution 从 Pending 到 Queued、Running、Success、PartialSuccess、Failed、Timeout、Canceled。
- 终态不能重新 Running。

### Manifest

覆盖：

- manifest 正常解析。
- manifest 缺少必要字段。
- command 非法。
- checksum/signature 字段解析。
- 新协议和兼容协议。

### Runtime

覆盖：

- 插件正常返回。
- 插件返回失败。
- 插件输出非法 JSON。
- 插件超时。
- stdout/stderr 限制。
- 输出清洗。
- secret/env 策略拒绝。
- container/process runner contract。

### 幂等

覆盖：

- 相同 key + 相同输入 + 相同插件返回同一 execution。
- 相同 key + 不同输入返回 `IDEMPOTENCY_CONFLICT`。
- 并发提交只创建一个逻辑任务的设计语义。

### Worker

覆盖：

- worker 消费任务。
- worker 写 attempt。
- worker 保存结果。
- worker 聚合最终状态。
- worker handler 失败时进入 retry/nack 语义。

### Webhook

覆盖：

- webhook 创建、启用、禁用。
- webhook HMAC 签名。
- webhook retry/backoff/DLQ baseline。
- tenant/project scope 隔离。
- SSRF 防护 baseline。

### Policy / Secret / Artifact

覆盖：

- RBAC 决策。
- scope 拒绝。
- secret resolver 不跨 tenant/project。
- encrypted secret provider。
- artifact store hash/URI 语义。

## API 测试重点

覆盖：

- `/api/v1/plugins/reload`
- `/api/v1/plugins`
- `/api/v1/plugins/{id}/enable`
- `/api/v1/executions`
- `/api/v1/executions/{id}`
- `/api/v1/executions/{id}/results`
- `/api/v1/executions/{id}/summary`
- `/api/v1/executions/{id}/events`
- `/api/v1/executions/{id}/attempts`
- `/api/v1/audit/executions/{id}`
- `/metrics`
- `/livez`、`/readyz`、`/workerz`、`/dependencyz`

## 闭环验证

`make verify` 是最重要的验收命令。

它会自动启动 server，并验证：

- 健康检查。
- 插件 reload。
- 插件 enable。
- 成功任务。
- 幂等重复提交。
- 幂等冲突。
- results。
- summary。
- events。
- attempts。
- audit。
- PartialSuccess。
- 分页。
- metrics。

## 生产化测试建议

当前发布版已完成 local/dev 闭环验证。生产化阶段应补充：

- 真实 Postgres 集成测试。
- Postgres migration 回归测试。
- SKIP LOCKED lease reclaim 测试。
- worker kill/restart 测试。
- server restart recovery 测试。
- tenant/project 数据隔离系统测试。
- container runner 安全策略测试。
- OTel trace 链路测试。
- release checksum/SBOM/signing 测试。
