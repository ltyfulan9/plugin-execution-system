# plugin-runtime-reviewer

用于检查插件执行隔离。

重点检查：

- 插件是否通过 manifest 注册。
- Runtime 是否使用 context timeout。
- 插件 stdout 是否必须是 JSON。
- success=false 是否被识别为插件失败。
- 非 JSON stdout 是否标记 InvalidOutput。
- 进程超时是否标记 Timeout。
- 单插件失败是否不影响其他插件。
- stderr 是否只保存 preview。
- command/workdir 是否存在逃逸风险。
