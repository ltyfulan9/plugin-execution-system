# Roadmap

## Current Release Focus

当前发布重点是闭环可验证：插件发现、插件管理、execution 创建、worker 执行、runtime 隔离、结果聚合、事件/attempt/audit 查询和 `make verify` 一键验收。

## Next Production Hardening

### Metadata and Queue

- 正式接入 pgx/Postgres。
- 增加 Postgres integration test。
- 验证 migration。
- 验证 unique idempotency constraint。
- 验证 `FOR UPDATE SKIP LOCKED` lease。
- 验证 heartbeat、visibility timeout、reclaim、retry、DLQ。

### Isolation

- 将 container runner 作为推荐运行时。
- 强化网络策略。
- 强化 env allowlist。
- 强化 mount allowlist。
- 强化 resource limit。
- 支持 image digest pinning。

### Security

- 接入 OIDC/SAML/mTLS identity provider。
- 接入 OPA/Rego policy engine。
- 接入 Vault/KMS secret provider。
- 增加 release signature、SBOM、attestation。

### Observability

- 接入 OpenTelemetry。
- 增加 API -> queue -> worker -> runtime -> result -> webhook trace。
- 增加 Prometheus 指标文档和 dashboard 示例。

### Artifact

- 增加 S3/MinIO artifact adapter。
- 大结果、大日志进入 object storage。
- DB 只保存 URI/hash/size metadata。

### Deployment

- 增加 Kubernetes manifests。
- 增加 Helm chart。
- 增加 backup/restore 文档。
- 增加 rolling upgrade 文档。

## Long-term Ecosystem

- 插件 registry。
- 插件版本兼容检查。
- 插件签名和信任策略。
- 插件 marketplace baseline。
- SDK 稳定化。
- Webhook replay UI。
