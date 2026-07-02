# Review Checklist

## 一键验收

- [ ] `go test ./...` 通过。
- [ ] `make verify` 通过。
- [ ] 输出包含 `VERIFICATION PASSED`。
- [ ] README 中的快速验证命令可执行。

## 题目核心要求

- [ ] 主语言是 Go。
- [ ] 主程序不依赖具体插件实现。
- [ ] 没有直接使用 Go 标准库 `plugin` 包作为核心方案。
- [ ] 没有直接接入现成插件系统或流程框架代替自身实现。
- [ ] 插件有 manifest 规范。
- [ ] 插件可以被扫描、注册、启用、禁用。
- [ ] 插件失败不会导致主服务崩溃。
- [ ] 有完整执行任务闭环。
- [ ] 有 README、测试说明和可运行验证脚本。

## 架构分层

- [ ] `cmd/server/main.go` 只做初始化和启动。
- [ ] handler 不直接访问数据库或文件。
- [ ] service 负责业务规则和状态流转。
- [ ] repository 只负责数据访问。
- [ ] model 只定义结构体和状态常量。
- [ ] worker 只负责异步消费和调度。
- [ ] runtime 只负责插件执行隔离。
- [ ] response 统一 JSON 响应和错误码。
- [ ] router 只注册路由。

## 闭环完整性

- [ ] server 可以启动。
- [ ] health/ready/worker/dependency 检查可用。
- [ ] 插件 reload 成功。
- [ ] 插件 enable 成功。
- [ ] 成功 execution 可以完成。
- [ ] 多插件 PartialSuccess 可以完成。
- [ ] results 可查询。
- [ ] summary 可查询。
- [ ] events 可查询。
- [ ] attempts 可查询。
- [ ] audit 可查询。
- [ ] metrics 可查询。

## 幂等和一致性

- [ ] 相同 `Idempotency-Key` 重复提交返回同一 execution。
- [ ] 相同 key 但不同输入返回 `IDEMPOTENCY_CONFLICT`。
- [ ] execution 状态变化有 event。
- [ ] worker 执行有 attempt。
- [ ] 失败插件不会覆盖成功插件结果。

## 安全边界

- [ ] 插件 command 有 allowlist。
- [ ] process runner 不允许未声明的敏感 env。
- [ ] 插件输出有限制。
- [ ] stdout/stderr 会清洗控制字符。
- [ ] 错误响应不暴露服务器绝对路径。
- [ ] webhook 有 HMAC 签名 baseline。
- [ ] tenant/project scope 模型存在。
- [ ] policy、secret、artifact plane 有清晰接口边界。

## 文档质量

- [ ] README 第一屏能说明项目是什么。
- [ ] README 有快速运行和验证命令。
- [ ] VERIFY.md 能解释验证脚本做了什么。
- [ ] API_SPEC.md 只以 `/api/v1` 作为未来接口主线。
- [ ] PLUGIN_SPEC.md 能说明插件如何编写。
- [ ] STATUS.md 不夸大未完成的生产能力。
- [ ] 不再用历史版本号作为发布定位。

## 不应该做的事

- [ ] 不把 local-json 描述成生产 metadata store。
- [ ] 不把 process runner 描述成企业推荐 runtime。
- [ ] 不宣称 OIDC/SAML、OPA、Vault、S3、完整 OTel 已经生产完成。
- [ ] 不继续堆无关 CRUD。
- [ ] 不让文档主线被历史版本记录污染。
