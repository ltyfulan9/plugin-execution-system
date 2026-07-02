# security-threat-review

用于检查安全边界。

重点检查：

- 管理员接口是否被 admin 权限保护。
- 普通用户是否只能看自己的 execution。
- response 是否泄露 token、绝对路径、完整 stderr。
- 插件是否能直接操作主系统数据。
- command 是否需要白名单。
- JSON 持久化是否存在并发写损坏风险。
- Running 任务取消是否需要更明确的语义说明。
