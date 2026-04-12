# Direct / WorkerPool 高 QPS 总结

状态：已完成（与当前实现与内置压测工具一致）

本文基于本仓库的实现（`server.AdServer`）与内置的 in-process 压测工具（`cmd/loadtest` / `go test -bench`），总结 Direct（进程内 Executor）与 WorkerPool（有界队列 + 固定 worker）两种并发调度方式的优缺点与容量估算方法，并给出压测落地建议。

## 结论先行

- 想在 **P99 < 50ms** 的同时冲非常高 QPS，关键不在平均延迟，而在 **排队控制 + 抖动控制（GC/调度/分配/锁）**。
- 在高 QPS 稳态运行下，**WorkerPool/有界执行器更容易把尾延迟控制住**（背压清晰、goroutine 数量稳定）。
- **百万 QPS 并非纯理论不可达**，但对每请求 CPU 时间、内存分配、payload 大小、网络栈开销的要求极端苛刻；在常规 `net/http + JSON` 路径下，更现实的稳定区间通常是 **几十万 QPS量级**（具体取决于 payload/分配/是否 TLS/sidecar 等）。本仓库内置压测为 in-process（不含 HTTP/JSON），因此测出来的数值只适用于“业务与并发模型”对比，不等同于端到端 HTTP QPS。

## Direct vs WorkerPool：优缺点

### Direct（进程内 Executor：goroutine + semaphore 限流）

- 优点
  - 低/中负载下通常延迟更低（排队少）。
  - 代码直观：任务直接并发，靠并发度限制器兜底。
- 缺点（高 QPS 关键）
  - 任务粒度很细时会产生大量 goroutine churn（创建/调度/栈增长），调度器开销明显。
  - 短生命周期对象多 → GC 压力上升 → P99 抖动更明显。
  - 若并发上限/背压策略不清晰，容易在流量尖峰出现“堆积 + 抖动”。

### WorkerPool（有界队列 + 固定 worker goroutines）

- 优点（高 QPS 关键）
  - goroutine 数量稳定（复用 worker），调度开销更可控。
  - 天然背压：队列满可快速拒绝/降级，利于把 P99 控在目标内。
  - 更适合稳态高吞吐场景。
- 缺点
  - 饱和时会排队，P99 很容易被“队列等待”拉高。
  - 队列策略不当会产生 head-of-line blocking（例如慢任务拖住快任务）。

## 对应到当前代码的“背压/排队”开关

- 请求级别在途上限（网关模型）：`Config.MaxConcurrentRequests`
  - `RequestAdmissionReject`：超限立即返回 `server: overloaded`（快速失败，保护尾延迟）
  - `RequestAdmissionBlock`：超限进入等待队列，直到获取 slot 或请求超时（更贴近网关排队）
- WorkerPool 级别背压：`infra.WorkerPool` 的 `queueSize` + `numWorkers`
  - 队列满时 `Submit` 直接失败，最终表现为请求失败（pool full / rejected）

## 容量估算：两个硬约束

### 1) CPU 预算（CPU-bound 近似）

当没有外部 RPC 时，上限常由 CPU/分配/调度/编解码决定。粗略估算：

> QPS_max ≈ (CPU 核心数 × 1s) / 每请求 CPU 时间

对 64 核：

- 1,000,000 QPS ⇒ 每请求 CPU 预算约 **64µs**
- 500,000 QPS ⇒ **128µs**
- 200,000 QPS ⇒ **320µs**

说明：

- 上述预算需要覆盖：HTTP 解析/路由、JSON 编解码、业务逻辑、runtime/GC、指标/日志等。
- 现实中通常要乘一个折损系数（例如 0.3~0.7），因为 GC/网络/调度会额外占用 CPU 并带来抖动。

### 2) 并发在途（Little’s Law）

> 在途并发 ≈ QPS × 延迟

如果目标 **P99=50ms**：

- 1,000,000 QPS ⇒ 在途并发约 **50,000**

这会显著推高内存占用（请求对象、缓冲、栈等）与 GC/调度成本。为了让 P99 稳定低于 50ms，通常需要：

- 在接近饱和前就开始 **限流/快速拒绝**，避免排队时间累积到尾延迟。

## 对当前实现的关键提醒（影响上限的点）

- 当前编排为“两阶段 + 扁平化 candidate 任务”。高 QPS 下最危险的是 **任务数量爆炸**：
  - 若每请求平均 `I` 个 imp、每个 imp `C` 个 candidate，则候选评估任务数约 `I*C`。
  - 高吞吐场景下，任务调度开销可能超过业务逻辑本身。
- 若要冲极限吞吐并稳定 P99：
  - 尽量减少任务粒度：例如“每 imp 一个任务，imp 内顺序处理 candidate”，降低调度与分配。
  - 明确背压：WorkerPool 队列必须有界，满了要有清晰策略（HTTP 429/503 或降级）。
  - 控制分配：JSON 编解码与临时对象分配往往是 64µs 预算里最大的敌人。

## 推荐的压测落地方法（获得真实 QPS 上限）

- 本仓库提供两套 in-process 工具用于对比并发模型：
  - Benchmark：`go test ./server -run '^$' -bench 'BenchmarkHandleRequest_' -benchmem`
  - Loadtest：`go run ./cmd/loadtest ...`，支持 `-sweep` 输出 CSV 与曲线点（success_rate/ok_rps/p95）
- 若要获得端到端真实上限（含 HTTP/JSON/网络/TLS/sidecar），再使用 **wrk2（恒定速率）** 或 vegeta 压测你自己的 HTTP 服务实现，关注：
  - 吞吐、P50/P90/P99、错误率（429/503/timeout）
  - pprof：CPU/allocs/heap/GC pause
- 分组对比：
  - Direct（进程内 Executor） vs WorkerPool（有界队列）
  - “每 candidate 一个任务” vs “每 imp 一个任务（imp 内顺序）”
- 对稳定性目标（P99<50ms）建议：
  - 设定明确的并发上限、队列容量、超时与拒绝策略
  - 压测时逐步升压，找到“刚好不爆 P99”的稳定点作为上限参考
