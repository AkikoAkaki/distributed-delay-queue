---
description: 验证 Watchdog 看门狗恢复机制
---

# Watchdog 验证流程

本流程用于验证 Watchdog 能否在 Worker 卡死/崩溃时，自动将未完成的任务重新放回队列。

## 核心机制

- **visibility_timeout**: 5秒 - Worker 拿到任务后，如果 5 秒内没有 ACK，任务会被 Watchdog 重新放回队列
- **watchdog_interval**: 5秒 - Watchdog 每 5 秒扫描一次超时任务
- **模拟场景**: Worker 处理任务时卡死 10 秒（超过 visibility_timeout）

## 验证步骤

### 1. 确认配置正确

检查 `config/config.yaml`:
```yaml
queue:
  visibility_timeout: 5  # 5 秒没处理完，就认为 Worker 挂了
  watchdog_interval: 5   # 每 5 秒检查一次
  max_retries: 3         # 默认重试 3 次
```

### 2. 启动 Server（如果还没启动）

// turbo
```bash
make run-server
```

**预期输出**:
```
Watchdog started. Interval: 5s, Timeout: 5s, MaxRetries: 3
Server listening on :9090
```

### 3. 启动 Worker（在新终端）

// turbo
```bash
make run-worker
```

**预期输出**:
```
Worker started, polling for tasks...
```

### 4. 发送测试任务（在第三个终端）

// turbo
```bash
go run scripts/test_client/main.go
```

**预期输出**:
```
Sending task (delay 5s)...
Task sent!
```

### 5. 观察 Worker 行为

**Worker 终端应该显示**:
```
--- Processed 1 tasks ---
[EXECUTE] TaskID: xxx, Payload: {"user": "alice"}, Delay: 0s
[SIMULATE] Worker is stuck for 10 seconds (simulating crash)...
```

此时 Worker 会卡住 10 秒。

### 6. 观察 Server 端 Watchdog 日志

**在 Worker 卡住约 5-10 秒后，Server 终端应该显示**:
```
[Watchdog] Recovered X tasks from processing queue
```

这说明 Watchdog 检测到任务超时（5秒），并将其重新放回了 `default` 队列。

### 7. 选项 A - 让 Worker 自然恢复

如果你等待 10 秒，Worker 会自动恢复并再次拉取到同一个任务：
```
[SIMULATE] Worker recovered (this should NOT appear if killed)
--- Processed 1 tasks ---
[EXECUTE] TaskID: xxx, Payload: {"user": "alice"}, Delay: 0s
[SIMULATE] Worker is stuck for 10 seconds (simulating crash)...
```

### 7. 选项 B - 模拟 Worker 崩溃（推荐）

**在 Worker 打印 `[SIMULATE] Worker is stuck...` 后立即按 `Ctrl+C` 杀掉 Worker**

然后重新启动 Worker:
```bash
make run-worker
```

**预期**: Worker 会再次收到同一个任务（因为 Watchdog 已经把它捞回队列了）

### 8. 验证重试次数

如果你重复杀掉 Worker 3 次（max_retries = 3），任务会进入死信队列（DLQ）。

**Server 日志会显示**:
```
[Watchdog] Task xxx exceeded max retries, moved to DLQ
```

## 成功标志

✅ Worker 卡住 5 秒后，Server 的 Watchdog 日志显示恢复了任务
✅ 重启 Worker 后能再次收到同一个任务
✅ 超过重试次数后，任务进入 DLQ

## 清理

测试完成后，记得移除模拟代码：
- 删除 `cmd/worker/main.go` 中的 `time.Sleep(10 * time.Second)` 和相关日志
