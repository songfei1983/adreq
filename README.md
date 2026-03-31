# adreq

[English](file:///Users/songfei/Develop/github.com/songfei1983/adreq/README.en.md)

一个用 Go 编写的广告竞价请求（ad request）处理链路示例工程：将一个 `BidRequest`（包含多个 `Imp`）并发处理为 `BidResponse`，中间包含候选生成、过滤链、出价、竞价排序与去重等步骤。

## 代码结构

- 入口与演示：[main.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/main.go)
- 应用服务（请求编排与响应组装）：[server/ad_server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/ad_server.go)
- 领域服务（单候选业务处理）：[bidder/processor.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/processor.go)
- 候选源（副作用）：[bidder/random_candidate_source.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/random_candidate_source.go)
- 并发执行器：[executor/executor.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/executor/executor.go)
- 基础设施（WorkerPool）：[bidder/worker_pool.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/worker_pool.go)
- 基础设施（预算存储）：[infra/budget_store.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/infra/budget_store.go)
- 策略/规则（过滤链、各类过滤器）：[filter/chain.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/filter/chain.go)
- 领域模型（请求/响应/中间对象，按 struct 拆分）：[model](file:///Users/songfei/Develop/github.com/songfei1983/adreq/model/)
- 文档索引（中英双语）：[docs/README.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/README.md)
- 架构图： [architecture.zh.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/architecture.zh.md) | [architecture.en.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/architecture.en.md)
- 时序图： [sequence.zh.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/sequence.zh.md) | [sequence.en.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/sequence.en.md)

## 核心流程

1. `server.AdServer` 接收 `BidRequest`
2. 对每个 `Imp` 通过 `CandidateSource` 拉取候选（由 Executor 并发调度）
3. 对每个候选 `CandidateAd` 调用 `ImpProcessor.ProcessCandidate`（过滤链 → 出价）得到决策
4. `BidFinalizer` 对每个 `Imp` 取 TopN 并执行预算扣减
5. 组装为 `BidResponse.SeatBid`

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

当前测试主要覆盖 server 的基本处理流程与并发场景：[server/server_test.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/server_test.go)

## 说明与已知问题

- 本工程以演示链路与并发模型为主，过滤器逻辑整体偏示例/占位实现。
- `HandleRequestWithPool` 会等待所有 worker 完成后再汇总结果，避免因为并发时序导致 `seatbid` 为空（见 [server/ad_server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/ad_server.go)）。
