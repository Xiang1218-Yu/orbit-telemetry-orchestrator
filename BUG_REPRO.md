# Bug 复现说明

## Bug 是什么
多个事件订阅者处理同一事件时，一个订阅者对嵌套设备信息的修改会影响另一个订阅者看到的内容。

## 如何触发
发布带有嵌套设备信息的事件，让一个订阅者修改收到的载荷，再检查另一个订阅者收到的原始内容。

## 运行指令
```bash
go test -v -count=1 ./internal/events -run '^TestBug004EventPayloadIsolationNestedState$'
```

## 错误信息
第二个订阅者观察到已被其他订阅者修改的载荷。

## 错误堆栈
```text
=== RUN   TestBug004EventPayloadIsolationNestedState
    bug004_event_payload_isolation_test.go:24: subscriber payload was mutated through another subscriber: changed
--- FAIL: TestBug004EventPayloadIsolationNestedState (0.00s)
FAIL
FAIL	orbit-telemetry-orchestrator/internal/events	0.880s
FAIL
```
