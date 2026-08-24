# Bug 复现说明

## Bug 是什么
同一设备在异常高峰期间同时收到多条异常时，开放事件中只保留了部分异常，后续证据和处置记录也不完整。

## 如何触发
并行提交同一设备和信号的多条异常，检查最终开放事件中的异常数量是否包含全部输入。

## 运行指令
```bash
go test -race -count=20 -v ./internal/service -run '^TestBug009IncidentCorrelationConcurrentAnomalies$'
```

## 错误信息
并发异常关联场景中的事件异常数量小于预期，单个异常场景仍可通过。

## 错误堆栈
```text
=== RUN   TestBug009IncidentCorrelationConcurrentAnomalies
    bug009_incident_correlation_test.go:51: incident anomaly count = 18, want 65
--- FAIL: TestBug009IncidentCorrelationConcurrentAnomalies (0.00s)
FAIL
FAIL	orbit-telemetry-orchestrator/internal/service	0.954s
FAIL
```
