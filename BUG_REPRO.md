# Bug 复现说明

## Bug 是什么
并行接收多路设备遥测并生成事件时，可能出现重复事件编号，后续记录会互相覆盖。

## 如何触发
在含有问题的版本中，并行执行事件编号生成场景，并观察重复编号和测试失败。

## 运行指令
```bash
go test -race -count=20 -v ./internal/service -run '^TestBug001ConcurrentIdsUnique$'
```

## 错误信息
并行编号场景失败，单线程顺序编号场景仍可通过。

## 错误堆栈
```text
WARNING: DATA RACE
Write at 0x00c0001f01d0 by goroutine 18:
  orbit-telemetry-orchestrator/internal/service.(*App).nextID()
      /private/tmp/orbit-repro-001.BPTy9q/internal/service/telemetry.go:123 +0x58
  orbit-telemetry-orchestrator/internal/service.TestBug001ConcurrentIdsUnique.func1()
      /private/tmp/orbit-repro-001.BPTy9q/internal/service/bug001_concurrent_ids_test.go:22 +0xb0

Previous read at 0x00c0001f01d0 by goroutine 20:
  orbit-telemetry-orchestrator/internal/service.(*App).nextID()
      /private/tmp/orbit-repro-001.BPTy9q/internal/service/telemetry.go:121 +0x40
  orbit-telemetry-orchestrator/internal/service.TestBug001ConcurrentIdsUnique.func1()
      /private/tmp/orbit-repro-001.BPTy9q/internal/service/bug001_concurrent_ids_test.go:22 +0xb0

    bug001_concurrent_ids_test.go:32: duplicate generated ID: incident-1
    testing.go:1712: race detected during execution of test
--- FAIL: TestBug001ConcurrentIdsUnique (0.01s)
FAIL
FAIL	orbit-telemetry-orchestrator/internal/service	0.654s
FAIL
```
