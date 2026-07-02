# Project Status

## 当前状态

项目处于可演示、可测试、闭环可验证的发布状态。

当前重点不是继续增加复杂平台功能，而是确保插件化任务执行系统的主链路可以稳定运行、可以被测试证明、可以被评审快速理解。

## 已完成能力

### 插件系统

- 支持 `plugins/` 目录扫描。
- 支持 manifest 解析。
- 支持插件 reload。
- 支持插件启用和禁用。
- 支持插件错误状态。
- 支持示例插件：`echo`、`text_stats`、`json_pick`、`error_demo`。
- 主程序不 import 插件实现，插件通过协议被调用。

### 执行系统

- 支持创建 execution。
- 支持 `Idempotency-Key`。
- 支持 Pending、Queued、Running、Success、PartialSuccess、Failed、Timeout、Canceled 等状态。
- 支持 worker 异步执行任务。
- 支持 runtime 隔离插件执行。
- 支持单插件失败不影响整个服务。
- 支持结果聚合 summary。
- 支持 results、events、attempts、audit 查询。

### API 和前端

- API 统一使用 `/api/v1` 作为主命名空间。
- 所有列表接口支持分页。
- 所有错误使用稳定错误码。
- 提供 `docs/openapi.json`。
- 提供前端演示页面。
- 提供 Go SDK、Python SDK 和 `pesctl` CLI baseline。

### 观测和验证

- 支持 `/livez`、`/readyz`、`/workerz`、`/dependencyz`。
- 支持 `/metrics`。
- 支持 `/debug/vars`。
- 支持 request_id 和 trace_id。
- 支持结构化日志 baseline。
- 提供 `make verify` 一键闭环验证。

### 企业架构边界

- 已定义 tenant/project scope 模型。
- 已定义 metadata store / durable queue contract。
- 已提供 Postgres schema 和 adapter baseline。
- 已定义 process/container/wasm/remote runner contract。
- 已定义 policy engine 边界。
- 已定义 secret reference 与 runtime injection 边界。
- 已定义 artifact store 边界。
- 已定义 webhook HMAC、retry、DLQ baseline。
- 已提供 release supply chain 文档和脚本 baseline。

## 已验证结果

已在当前包内运行：

```bash
go test ./...
make verify
```

`make verify` 会启动临时 dev server 并验证完整闭环。验证通过时输出：

```text
VERIFICATION PASSED
```

## 本地模式说明

local-json 只用于本地开发和评审验证，不作为正式生产路线。

启动 local/dev mode 必须显式声明：

```bash
APP_MODE=dev
METADATA_STORE=local-json
ALLOW_LOCAL_JSON_STORE=true
```

正式生产应使用 Postgres/HA metadata store 和 durable execution 后端。

## 当前不宣称完成的内容

以下内容已有接口、schema、contract 或 baseline，但不应宣称为完全生产 GA：

- 真实 pgx/Postgres 集成测试全覆盖。
- 完整 OIDC/SAML 企业认证。
- 完整 OPA/Rego runtime。
- 完整 Vault/KMS secret provider。
- 完整 S3/MinIO artifact adapter。
- 完整 OpenTelemetry trace pipeline。
- 完整 Sigstore/SBOM/attestation 发布流水线。
- 完整 Kubernetes/Helm 运维交付。

## 发布策略

当前发布版适合用于：

- 笔试项目提交。
- 插件化执行系统演示。
- 后端工程分层展示。
- 企业级架构能力展示。
- 后续生产化改造的基础版本。

不建议继续无节制增加模块。下一阶段应围绕真实生产验证推进：Postgres 集成测试、durable queue 故障恢复测试、tenant/project 隔离测试、container runner 强化。
