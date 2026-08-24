# Bug 复现说明

## Bug 是什么
服务停止时，队列中正在执行的遥测评估任务没有及时结束，停止流程会等待任务完成。

## 如何触发
启动后台队列并提交一个正在等待停止信号的任务，然后停止队列并观察任务是否收到取消通知。

## 运行指令
```bash
go test -v -count=1 ./internal/queue -run '^TestBug002WorkerContextPropagation$'
```

## 错误信息
停止队列后，正在执行的任务仍未收到取消信号。

## 错误堆栈
```text
=== RUN   TestBug002WorkerContextPropagation
    bug002_worker_context_test.go:32: worker did not receive queue cancellation
--- FAIL: TestBug002WorkerContextPropagation (0.30s)
FAIL
FAIL	orbit-telemetry-orchestrator/internal/queue	2.396s
FAIL
```
