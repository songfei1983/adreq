# adreq

[English](README.en.md)

一个用 Go 编写的广告竞价请求（ad request）处理链路示例工程：将一个 `BidRequest`（包含多个 `Imp`）并发处理为 `BidResponse`，中间包含候选生成、过滤链、出价、竞价排序与去重等步骤。

## 代码结构

- 入口与演示：[main.go](main.go)
- HTTP 服务入口（K8s 部署目标）：[cmd/api](cmd/api/)
- 应用服务（请求编排与响应组装）：[server/ad_server.go](server/ad_server.go)
- 领域服务（单候选业务处理）：[bidder/processor.go](bidder/processor.go)
- 候选源（副作用）：[infra/random_candidate_source.go](infra/random_candidate_source.go)
- 并发执行器：[executor/executor.go](executor/executor.go)
- 基础设施（WorkerPool）：[infra/worker_pool.go](infra/worker_pool.go)
- 基础设施（预算存储）：[infra/budget_store.go](infra/budget_store.go)
- 策略/规则（过滤链、各类过滤器）：[filter/chain.go](filter/chain.go)
- 领域模型（请求/响应/中间对象，按 struct 拆分）：[model](model/)
- 文档索引（中英双语）：[docs/README.md](docs/README.md)
- 架构图： [architecture.zh.md](docs/architecture.zh.md) | [architecture.en.md](docs/architecture.en.md)
- 时序图： [sequence.zh.md](docs/sequence.zh.md) | [sequence.en.md](docs/sequence.en.md)

## 核心流程

1. `server.AdServer` 接收 `BidRequest`
2. 对每个 `Imp` 通过 `CandidateSource` 拉取候选（由 Executor 并发调度）
3. 对每个候选 `CandidateAd` 调用 `ImpProcessor.ProcessCandidate`（过滤链 → 出价）得到决策
4. `BidFinalizer` 对每个 `Imp` 取 TopN 并执行预算扣减
5. 组装为 `BidResponse.SeatBid`

## 约定

- 业务逻辑（Business Logic）与并发调度（Concurrency Orchestration）完全解耦。
- 业务逻辑保持“纯净、可测试”：
  - 不包含 `goroutine` / `channel` / `sync.*` / `select` 等并发元素
  - 只处理单个任务：如单候选处理 `ImpProcessor.ProcessCandidate`
- 并发调度由 Executor 统一管理（类似 Java 的 `ExecutorService`，但 Go idiomatic）：
  - 负责任务提交、并发度控制、错误聚合、`context` 生命周期管理
  - 避免深层嵌套 goroutine：所有并发启动点集中在 Executor 内部
- 副作用（I/O、随机、延迟、状态存储、扣减等）外移到基础设施层（`infra`），通过接口注入到编排层（`server`）。
- 可测试性优先：
  - `ExecutorFactory` / `BidFinalizer` 可注入替身（fake/mock），从而让编排层可在“串行执行”下稳定单测
  - 业务逻辑可独立单测，不依赖并发或运行时组件

## 运行

```bash
go run .
```

运行后会依次执行两种模式并输出 JSON：

- Direct Processing：`AdServer` 通过进程内 Executor 做有界并发调度
- Worker Pool Processing：编排模型一致，但任务提交到有界的 `WorkerPool`

## 测试

```bash
go test ./...
```

当前测试主要覆盖 server 的基本处理流程与并发场景：[server/server_test.go](server/server_test.go)

## 性能测试

基准测试（Direct vs WorkerPool，对比 ns/op、allocs/op、%rejected）：见 [server/bench_test.go](server/bench_test.go)

```bash
go test ./server -run '^$' -bench 'BenchmarkHandleRequest_' -benchmem
```

负荷测试（in-process，不走 HTTP）：见 [cmd/loadtest](cmd/loadtest/)

```bash
go run ./cmd/loadtest -mode=pool -sweep -duration=2s -imps=6 -cands=40 \
  -concurrency-list=32,64,128 -pool-workers-list=16 -pool-queue-list=64,1024,64k
```

入口限流（最大并发在途请求）：

- `-max-requests=N` 开启请求级别并发上限
- `-admission=reject|block` 控制超限策略：快速拒绝或排队等待直到超时

## 说明与已知问题

- 本工程以演示链路与并发模型为主，过滤器逻辑整体偏示例/占位实现。
- `HandleRequestWithPool` 会等待所有 worker 完成后再汇总结果，避免因为并发时序导致 `seatbid` 为空（见 [server/ad_server.go](server/ad_server.go)）。
