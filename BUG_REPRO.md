# Bug 复现说明

## Bug 是什么
轻量部署未配置审计组件时，访问审计查询接口返回内部错误，而就绪检查仍然可以正常返回。

## 如何触发
在不提供审计组件的服务配置下访问审计查询接口，观察响应状态和响应内容。

## 运行指令
```bash
go test -v -count=1 ./internal/httpapi -run '^TestBug007OptionalAuditMissingDependency$'
```

## 错误信息
审计查询返回 500 和 `internal server error`。

## 错误堆栈
```text
=== RUN   TestBug007OptionalAuditMissingDependency
2026/08/24 18:01:47 INFO http request method=GET path=/v1/audit duration=124.541µs
    bug007_optional_audit_test.go:18: audit status = 500, want 200; body=internal server error
--- FAIL: TestBug007OptionalAuditMissingDependency (0.00s)
FAIL
FAIL	orbit-telemetry-orchestrator/internal/httpapi	0.607s
FAIL
```
