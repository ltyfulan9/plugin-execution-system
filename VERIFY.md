# Closed-loop Verification

本文说明如何验证 Plugin Execution System 的完整业务闭环。

## 推荐命令

```bash
make verify
```

该命令会执行两步：

```bash
go test ./...
python3 scripts/verify_closed_loop.py
```

## 验证脚本做了什么

`scripts/verify_closed_loop.py` 会自动完成以下流程：

1. 创建临时 metadata 目录。
2. 以 local/dev mode 启动 server。
3. 检查服务存活。
4. 检查 ready、worker、dependency 状态。
5. 调用插件 reload。
6. 启用 `echo`、`text_stats`、`error_demo`。
7. 创建一个成功任务。
8. 使用相同 `Idempotency-Key` 重复提交，验证返回同一个 execution。
9. 使用相同 `Idempotency-Key` 但不同输入提交，验证返回 `IDEMPOTENCY_CONFLICT`。
10. 等待 worker 执行完成。
11. 查询 summary，确认成功任务状态为 `Success`。
12. 查询 results，确认每个插件结果正确。
13. 查询 events，确认状态事件完整。
14. 查询 attempts，确认 worker attempt 被记录。
15. 查询 audit，确认执行链路有审计记录。
16. 创建一个包含失败插件的任务。
17. 确认最终状态为 `PartialSuccess`。
18. 验证 execution 分页。
19. 验证 `/metrics` 包含关键指标。

通过时输出：

```text
VERIFICATION PASSED
```

## 手动验证步骤

启动服务：

```bash
APP_MODE=dev \
METADATA_STORE=local-json \
ALLOW_LOCAL_JSON_STORE=true \
SERVER_ADDR=:8080 \
go run ./cmd/server
```

检查健康状态：

```bash
curl http://127.0.0.1:8080/livez
curl http://127.0.0.1:8080/readyz
curl http://127.0.0.1:8080/workerz
curl http://127.0.0.1:8080/dependencyz
```

reload 插件：

```bash
curl -X POST \
  -H 'Authorization: Bearer admin-token' \
  http://127.0.0.1:8080/api/v1/plugins/reload
```

查看插件：

```bash
curl -H 'Authorization: Bearer admin-token' \
  'http://127.0.0.1:8080/api/v1/plugins?page=1&page_size=20'
```

创建任务时需要使用插件 ID。推荐直接使用 `make verify`，避免手动复制 ID。

## 验收结论

如果 `make verify` 通过，可以证明：

- 服务能启动。
- 插件能被发现和启用。
- execution 能被创建。
- worker 能执行任务。
- runtime 能调用插件。
- 成功和部分成功场景都能被正确聚合。
- 幂等和冲突语义正确。
- results、summary、events、attempts、audit、metrics 都可查询。
