# adreq

[中文](README.md)

A Go demo project for an ad-request (bid request) processing pipeline. It processes a `BidRequest` (with multiple `Imp`s) concurrently into a `BidResponse`, including candidate generation, filter chain, bidding, and top-N selection.

## Code Map

- Entry & demo: [main.go](main.go)
- HTTP API entry (for K8s): [cmd/api](cmd/api/)
- Application service (orchestration & response assembly): [server/ad_server.go](server/ad_server.go)
- Domain service (single-candidate business logic): [bidder/processor.go](bidder/processor.go)
- Candidate source (side-effects): [infra/random_candidate_source.go](infra/random_candidate_source.go)
- Concurrency executor: [executor/executor.go](executor/executor.go)
- Infrastructure (WorkerPool): [infra/worker_pool.go](infra/worker_pool.go)
- Infrastructure (budget store): [infra/budget_store.go](infra/budget_store.go)
- Policies/rules (filter chain, filters): [filter/chain.go](filter/chain.go)
- Domain model (request/response/intermediate structs, split by struct): [model](model/)
- Docs index (bilingual): [docs/README.md](docs/README.md)
- Architecture: [architecture.zh.md](docs/architecture.zh.md) | [architecture.en.md](docs/architecture.en.md)
- Sequence: [sequence.zh.md](docs/sequence.zh.md) | [sequence.en.md](docs/sequence.en.md)

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

Tests are mainly in: [server/server_test.go](server/server_test.go)

## Performance

Benchmarks (Direct vs WorkerPool, reports ns/op, allocs/op, %rejected): [server/bench_test.go](server/bench_test.go)

```bash
go test ./server -run '^$' -bench 'BenchmarkHandleRequest_' -benchmem
```

Load test (in-process, no HTTP): [cmd/loadtest](cmd/loadtest/)

```bash
go run ./cmd/loadtest -mode=pool -sweep -duration=2s -imps=6 -cands=40 \
  -concurrency-list=32,64,128 -pool-workers-list=16 -pool-queue-list=64,1024,64k
```

Request admission (cap in-flight requests):

- `-max-requests=N` enables request-level concurrency cap
- `-admission=reject|block` chooses overflow strategy: fast reject or wait-for-slot until timeout

## Notes

- This repository focuses on demonstrating the pipeline and concurrency model; filter implementations are mostly placeholders.
- `HandleRequestWithPool` waits for all workers to finish before aggregating results, avoiding empty `seatbid` due to timing issues (see [server/ad_server.go](server/ad_server.go)).
