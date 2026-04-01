# adreq

[中文](file:///Users/songfei/Develop/github.com/songfei1983/adreq/README.md)

A Go demo project for an ad-request (bid request) processing pipeline. It processes a `BidRequest` (with multiple `Imp`s) concurrently into a `BidResponse`, including candidate generation, filter chain, bidding, and top-N selection.

## Code Map

- Entry & demo: [main.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/main.go)
- Application service (orchestration & response assembly): [server/ad_server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/ad_server.go)
- Domain service (single-candidate business logic): [bidder/processor.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/bidder/processor.go)
- Candidate source (side-effects): [random_candidate_source.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/infra/random_candidate_source.go)
- Concurrency executor: [executor/executor.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/executor/executor.go)
- Infrastructure (WorkerPool): [worker_pool.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/infra/worker_pool.go)
- Infrastructure (budget store): [infra/budget_store.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/infra/budget_store.go)
- Policies/rules (filter chain, filters): [filter/chain.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/filter/chain.go)
- Domain model (request/response/intermediate structs, split by struct): [model](file:///Users/songfei/Develop/github.com/songfei1983/adreq/model/)
- Docs index (bilingual): [docs/README.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/README.md)
- Architecture: [architecture.zh.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/architecture.zh.md) | [architecture.en.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/architecture.en.md)
- Sequence: [sequence.zh.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/sequence.zh.md) | [sequence.en.md](file:///Users/songfei/Develop/github.com/songfei1983/adreq/docs/sequence.en.md)

## Pipeline

1. `server.AdServer` receives `BidRequest`
2. For each `Imp`, it fetches candidates via `CandidateSource` (concurrently via an Executor)
3. For each `CandidateAd`, it calls `ImpProcessor.ProcessCandidate` (filter chain → bidder) and gets a decision
4. `BidFinalizer` selects top-N bids per `Imp` and applies budget deduction
5. Assembles `BidResponse.SeatBid`

## Run

```bash
go run .
```

It runs two modes and prints JSON:

- Direct Processing: `AdServer` orchestrates work through an in-process Executor (bounded concurrency)
- Worker Pool Processing: same orchestration model, but tasks are submitted to a bounded `WorkerPool`

## Test

```bash
go test ./...
```

Tests are mainly in: [server/server_test.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/server_test.go)

## Notes

- This repository focuses on demonstrating the pipeline and concurrency model; filter implementations are mostly placeholders.
- `HandleRequestWithPool` waits for all workers to finish before aggregating results, avoiding empty `seatbid` due to timing issues (see [server/ad_server.go](file:///Users/songfei/Develop/github.com/songfei1983/adreq/server/ad_server.go)).
