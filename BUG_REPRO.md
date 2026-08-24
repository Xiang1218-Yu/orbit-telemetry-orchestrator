# Bug 复现说明

## Bug 是什么
运维页面重复提交同一个通知动作请求时，接口返回了新的动作编号，但该动作没有对应的有效持久化记录。

## 如何触发
使用同一个幂等请求标识连续创建通知动作，然后比较两次返回的动作编号和动作列表中的有效动作。

## 运行指令
```bash
go test -v -count=1 ./internal/service -run '^TestBug006ActionIdempotencyRepeatedRequest$'
```

## 错误信息
重复请求返回了不同的动作编号，后一次编号不是第一次请求已经保存的动作。

## 错误堆栈
```text
=== RUN   TestBug006ActionIdempotencyRepeatedRequest
    bug006_action_idempotency_test.go:36: repeated request returned "action-2", want durable action "action-1"
--- FAIL: TestBug006ActionIdempotencyRepeatedRequest (0.00s)
FAIL
FAIL	orbit-telemetry-orchestrator/internal/service	0.960s
FAIL
```
