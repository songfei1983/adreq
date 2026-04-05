# Sequence

This repository demonstrates two processing modes:

- Direct: `server.AdServer` uses an Executor to fan out tasks (bounded concurrency) for `Imp` candidate fetch and candidate evaluation; business logic stays synchronous per-task
- WorkerPool: same orchestration model, but the Executor submits tasks to a bounded `WorkerPool`

## Direct Processing

```mermaid
sequenceDiagram
  autonumber
  participant Caller as main/runDemo
  participant S as server.AdServer
  participant E as executor.Executor
  participant CS as CandidateSource
  participant P as ImpProcessor
  participant FC as filter.Chain
  participant F as filter.Filter*
  participant B as bidder.Bidder
  participant Fin as BidFinalizer

  Caller->>S: HandleRequest(ctx, BidRequest)
  opt MaxConcurrentRequests enabled
    S->>S: request admission (reject|block)
  end
  loop each Imp
    S->>E: Go(fetch candidates)
    E->>CS: FetchCandidates(reqCtx, imp)
    CS-->>E: []CandidateAd
    E-->>S: candidates
  end
  loop each CandidateAd
    S->>E: Go(process candidate)
    E->>P: ProcessCandidate(reqCtx, ad)
    P->>FC: Apply(ctx, ad)
    loop each filter
      FC->>F: Filter(ctx, ad)
    end
    alt passed
      P->>B: Bid(ctx, ad)
      B-->>P: Bid
      P-->>E: CandidateDecision
    else rejected
      P-->>E: nil
    end
    E-->>S: CandidateDecision
  end
  S->>Fin: Finalize(decisions, budget)
  Fin-->>S: []SeatBid
  S-->>Caller: BidResponse
```

## Worker Pool Processing

```mermaid
sequenceDiagram
  autonumber
  participant Caller as main/runDemo
  participant S as server.AdServer
  participant E as executor.Executor
  participant CS as CandidateSource
  participant WP as infra.WorkerPool
  participant W as worker goroutine
  participant P as ImpProcessor
  participant FC as filter.Chain
  participant B as bidder.Bidder
  participant Fin as BidFinalizer

  Caller->>S: HandleRequestWithPool(ctx, BidRequest)
  opt MaxConcurrentRequests enabled
    S->>S: request admission (reject|block)
  end
  loop each Imp
    S->>E: Go(fetch candidates)
    E->>WP: Submit(reqCtx, job)
  end
  loop each fetch job
    WP-->>W: dispatch job
    W->>CS: FetchCandidates(reqCtx, imp)
    CS-->>W: []CandidateAd
    W-->>S: candidates
  end
  loop each CandidateAd
    S->>E: Go(process candidate)
    E->>WP: Submit(reqCtx, job)
  end
  loop each candidate job
    WP-->>W: dispatch job
    W->>P: ProcessCandidate(reqCtx, ad)
    P->>FC: Apply(ctx, ad)
    P->>B: Bid(ctx, ad)
    P-->>W: CandidateDecision
    W-->>S: CandidateDecision
  end
  S->>Fin: Finalize(decisions, budget)
  Fin-->>S: []SeatBid
  S-->>Caller: BidResponse
```
