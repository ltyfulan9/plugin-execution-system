# Implementation Plan

## 当前目标

当前目标是保持项目轻量、闭环、可验证，不再继续无节制增加企业概念。

短期工作重点：

1. 保证 `make verify` 一键通过。
2. 保证 README、VERIFY、TEST_PLAN 与实际代码一致。
3. 保证插件主链路清晰可讲。
4. 保证提交包没有历史版本叙述污染。
5. 保证 local/dev mode 明确只是验证模式。

## 已完成主线

- 插件 manifest。
- 插件扫描与 reload。
- 插件状态管理。
- execution 创建。
- 幂等键处理。
- worker 异步执行。
- runtime 执行隔离。
- results 保存。
- summary 聚合。
- events / attempts / audit。
- webhook baseline。
- metrics baseline。
- 前端演示。
- CLI 和 SDK baseline。
- `make verify` 闭环验证。

## 不再优先做的内容

短期不再继续优先堆：

- 更多普通 CRUD。
- 更多示例插件。
- 前端美化。
- 大量未落地的企业扩展概念。
- 把 local-json 伪装成生产路线。

## 下一阶段建议

在当前闭环稳定后，再做生产化增强：

1. 引入正式 Postgres driver。
2. 增加真实 Postgres integration test。
3. 强化 tenant/project scope 测试。
4. 做 worker kill / lease reclaim 验证。
5. 强化 container runner。
6. 引入 S3/MinIO artifact adapter。
7. 引入 OTel trace pipeline。
8. 完成 release checksum/SBOM/signing。

## 提交策略

对外提交时只强调已经跑通和验证的内容：

- 插件机制。
- 任务执行闭环。
- 状态流转。
- 错误隔离。
- 结果聚合。
- 审计和事件。
- 一键验证。

企业扩展放在架构说明和路线规划中，不要当作已经完全生产落地的功能宣传。
