# RESEARCH_NOTES.md

## 调研结论

插件系统大致有三类实现方式：

1. 语言内动态加载。
2. 进程型插件。
3. 注册表 + hook / entry point 模式。

本项目最终选择进程型插件。

## Go 方向

Go 标准库 plugin 包可以动态加载 Go 插件，但存在平台和工程限制，不适合作为跨平台、可快速演示的项目基础。

HashiCorp go-plugin 使用独立进程 + RPC 的方式实现插件系统。Vault 插件架构也强调插件是独立应用，插件和主进程不共享内存，插件崩溃不应该导致主系统崩溃。

本项目不直接上 RPC/gRPC，而是用更轻量的 stdin/stdout JSON 协议模拟同类架构思想。

## Python 方向

pluggy 使用 PluginManager、hookspec、hookimpl 的模型，强调插件注册和 hook 调用。

stevedore 基于 entry points 管理动态扩展，强调插件发现、加载、启用方式和扩展管理模式。

本项目吸收这些思想，把插件 manifest、registry、status、reload 做成平台能力。

## 本项目方案

最终方案：

Go 主系统负责：

- manifest 扫描
- registry 同步
- 插件状态管理
- execution 任务编排
- runtime 执行隔离
- result 聚合
- audit 记录

插件负责：

- 接收 stdin JSON
- 执行独立数据处理逻辑
- 输出 stdout JSON

## 为什么不是 Go 原生 plugin

因为本项目需要：

- 尽量跨平台。
- 方便用 Python/Go/Node 写插件。
- 插件失败不影响主进程。
- 降低构建复杂度。
- 方便前端演示和测试。

进程型插件更合适。
