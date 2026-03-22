# adreq

一个用 Go 编写的广告竞价请求（ad request）处理链路示例工程：将一个 `BidRequest`（包含多个 `Imp`）并发处理为 `BidResponse`，中间包含候选生成、过滤链、出价、竞价排序与去重等步骤。

## 代码结构

- 入口与演示：[main.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/main.go)
- 请求编排层（按 Imp 并发处理、聚合竞价结果、生成响应）：[server/server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/server.go)
- 候选与竞价核心（候选生成、过滤链调用、出价、竞价排序/去重、并发限流）：[bidder/bidder.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/bidder.go)
- WorkerPool（固定 worker + 有界队列）：[bidder/workerpool.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/workerpool.go)
- 过滤链与预算缓存：[filter/filter.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/filter/filter.go)，过滤器实现：[filter/impl.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/filter/impl.go)
- 数据结构：[model/model.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/model/model.go)，候选对象：[model/candidate.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/model/candidate.go)

## 核心流程

1. `server.AdServer` 接收 `BidRequest`
2. 对每个 `Imp` 并发执行 `Processor.ProcessImp`
3. `ProcessImp` 生成候选（示例为随机模拟）→ 通过 `filter.Chain` 逐个过滤 → 调用 `Bidder.Bid` 出价
4. 将所有 bid 作为候选投入 `Auction`
5. `Auction` 按价格排序并进行去重（同一 `ImpID` 仅选一个 winner；同一 `(AdSlotID, AdvertiserID)` 只允许一次），输出 TopN
6. 组装为 `BidResponse.SeatBid`

## 运行

```bash
go run .
```

运行后会依次执行两种模式并输出 JSON：

- Direct Processing：每个 `Imp` 启 goroutine，imp 内部对候选并发（带 semaphore 限流）
- Worker Pool Processing：将每个 `Imp` 封装为 job 提交到固定 worker 的有界队列中处理

## 测试

```bash
go test ./...
```

当前测试主要覆盖 server 的基本处理流程与并发场景：[server/server_test.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/server_test.go)

## 说明与已知问题

- 本工程以演示链路与并发模型为主，过滤器逻辑整体偏示例/占位实现。
- `HandleRequestWithPool` 当前会在 worker 尚未完成时就从 `Auction` 取结果，可能导致 `seatbid` 为空；这是示例中的一个并发时序问题（见 [server/server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/server.go)）。
