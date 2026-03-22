# adreq

[中文](file:///Users/songfei/Develop/github.com/songfei1983/adreq/README.md)

A Go demo project for an ad-request (bid request) processing pipeline. It processes a `BidRequest` (with multiple `Imp`s) concurrently into a `BidResponse`, including candidate generation, filter chain, bidding, and top-N selection.

## Code Map

- Entry & demo: [main.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/main.go)
- Application service (orchestration & response assembly): [server/ad_server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/ad_server.go)
- Domain service (candidate processing, filtering, bidding, concurrency limiting): [bidder/processor.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/processor.go)
- Infrastructure (WorkerPool): [bidder/worker_pool.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/worker_pool.go)
- Policies/rules (filter chain, budget cache, filters): [filter/chain.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/filter/chain.go)
- Domain model (request/response/intermediate structs, split by struct): [model](file:///Users/songfei/Develop/github.com/songfei1983/adreq/model/)
- Docs index (bilingual): [docs/README.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/README.md)
- Architecture: [architecture.zh.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/architecture.zh.md) | [architecture.en.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/architecture.en.md)
- Sequence: [sequence.zh.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/sequence.zh.md) | [sequence.en.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/sequence.en.md)

## Pipeline

1. `server.AdServer` receives `BidRequest`
2. For each `Imp`, it calls `Processor.ProcessImp` concurrently
3. `ProcessImp` generates candidates (random simulation in this demo) → runs `filter.Chain` → calls `Bidder.Bid`
4. `server.AdServer` selects top-N bids per `Imp` by descending price (default: 2)
5. Assembles `BidResponse.SeatBid`

## Run

```bash
go run .
```

It runs two modes and prints JSON:

- Direct Processing: one goroutine per `Imp`; inside each `Imp`, candidates are processed concurrently (limited by a `semaphore`)
- Worker Pool Processing: wraps each `Imp` as a job and submits it to a bounded `WorkerPool`

## Test

```bash
go test ./...
```

Tests are mainly in: [server/server_test.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/server_test.go)

## Notes

- This repository focuses on demonstrating the pipeline and concurrency model; filter implementations are mostly placeholders.
- `HandleRequestWithPool` waits for all workers to finish before aggregating results, avoiding empty `seatbid` due to timing issues (see [server/ad_server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/ad_server.go)).

