# Bug 复现说明

## Bug 是什么
设备注册请求已经取消后，创建接口仍然返回成功，系统会留下不完整的设备记录。

## 如何触发
在设备注册操作完成前取消请求，然后检查返回错误、设备列表和同名设备是否还能再次注册。

## 运行指令
```bash
go test -v -count=1 ./internal/service -run '^TestBug010CanceledRequestStateNoPersistence$'
```

## 错误信息
取消的设备注册没有返回 `context canceled`。

## 错误堆栈
```text
=== RUN   TestBug010CanceledRequestStateNoPersistence
    bug010_canceled_request_state_test.go:19: CreateDevice error = <nil>, want context canceled
--- FAIL: TestBug010CanceledRequestStateNoPersistence (0.00s)
FAIL
FAIL	orbit-telemetry-orchestrator/internal/service	0.915s
FAIL
```
