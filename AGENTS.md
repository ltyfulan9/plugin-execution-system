# AGENTS.md

## 项目目标

把本项目做成一个企业级 Go 后端项目：插件化任务执行平台。不要写成 main.go 里扫描目录、调用脚本、直接返回结果的 demo。

核心闭环必须完整：

用户登录 -> 查看插件 -> 管理员 reload / enable / disable -> 用户创建执行任务 -> 幂等校验 -> 入队 -> worker 异步执行 -> runtime 进程隔离调用插件 -> 保存每个插件结果 -> 聚合任务状态 -> 前端/API 查看结果 -> 审计日志可追踪。

## 必须遵守的分层规则

main.go 只做初始化和启动。

router 只注册路由，不写业务规则。

handler 只做 HTTP 参数解析、权限上下文读取、调用 service、返回统一响应。

service 写业务规则、状态机、幂等、资源归属、结果聚合。

repository 只做数据访问，不判断业务状态。

model 只定义结构体、状态、常量，不写复杂流程。

runtime 只负责插件执行隔离，不关心 HTTP 和权限。

worker 只负责异步消费和并发控制，不解析 HTTP 请求。

response 只负责统一 JSON 和错误码。

middleware 只负责 request_id、日志、认证、panic recovery。

## 禁止事项

禁止 handler 直接操作 repository。

禁止 repository 判断插件是否能执行。

禁止把所有业务写在 service.go。

禁止把所有接口写在 handler.go。

禁止插件直接访问主系统数据库。

禁止插件 panic 或进程异常导致主服务崩溃。

禁止绕过状态机直接改任务状态。

禁止把插件完整 stderr、服务器绝对路径、token 等敏感信息返回给前端。

## 推荐开发方式

先保证主闭环跑通，再补细节。

开发顺序：

1. 能启动服务。
2. 能扫描插件。
3. 能登录。
4. 能 enable 插件。
5. 能创建 execution。
6. worker 能执行插件。
7. 能保存 result。
8. 能 summary。
9. 前端能演示。
10. go test ./... 通过。

## 可使用的本地 Agents / Skills

可以把 `.agents/skills/backend-test-runner` 用于测试闭环。

可以把 `.agents/skills/api-contract-tester` 用于 API 契约检查。

可以把 `.agents/skills/plugin-runtime-reviewer` 用于插件执行隔离检查。

可以把 `.agents/skills/security-threat-review` 用于安全边界检查。

## 交付标准

最终提交前必须满足：

- gofmt 已运行。
- go test ./... 通过。
- README 能指导从零启动。
- TEST_PLAN 覆盖异常场景。
- REVIEW_CHECKLIST 全部检查。
- STATUS 更新到当前真实状态。
