# api-contract-tester

用于检查前后端 API 契约一致性。

重点检查：

- README / API_SPEC 的路由是否和 internal/router 一致。
- web/static/app.js 调用的接口是否真实存在。
- 请求方法是否匹配。
- Authorization 和 Idempotency-Key 是否正确传递。
- 返回 JSON 是否走统一 response。
- 错误码是否在 response/codes.go 中定义。

发现问题后修复，并用 go test ./... 或 curl 手动流程验证。
