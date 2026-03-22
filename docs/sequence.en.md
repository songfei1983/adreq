# Sequence

This repository demonstrates two processing modes:

- Direct: one goroutine per `Imp`; inside each `Imp`, candidates are processed concurrently (limited by a `semaphore`)
- WorkerPool: each `Imp` is wrapped as a job submitted to `WorkerPool` and executed by fixed workers; inside each job, candidates are still processed concurrently

## Direct Processing

```mermaid
sequenceDiagram
  autonumber
  participant Caller as main/runDemo
  participant S as server.AdServer
  participant P as bidder.Processor
  participant FC as filter.Chain
  participant F as filter.Filter*
  participant B as bidder.Bidder

  Caller->>S: HandleRequest(ctx, BidRequest)
  loop each Imp
    par goroutine per Imp
      S->>P: ProcessImp(reqCtx, imp)
      P->>P: fetchCandidates()
      loop each CandidateAd
        par candidate goroutine (semaphore limited)
          P->>FC: Apply(ctx, ad)
          loop each filter
            FC->>F: Filter(ctx, ad)
          end
          alt passed
            P->>B: Bid(ctx, ad)
            B-->>P: Bid
          else rejected
            P-->>P: drop
          end
        end
      end
      P-->>S: []*Bid
      S-->>S: topBids + append SeatBid
    end
  end
  S-->>Caller: BidResponse
```

## Worker Pool Processing

```mermaid
sequenceDiagram
  autonumber
  participant Caller as main/runDemo
  participant S as server.AdServer
  participant WP as bidder.WorkerPool
  participant W as worker goroutine
  participant P as bidder.Processor
  participant FC as filter.Chain
  participant B as bidder.Bidder

  Caller->>S: HandleRequestWithPool(ctx, BidRequest)
  loop each Imp
    S->>WP: Submit(reqCtx, job)
  end
  loop each job
    WP-->>W: dispatch job
    W->>S: job(reqCtx)
    S->>P: ProcessImp(reqCtx, imp)
    P->>FC: Apply(ctx, ad)
    P->>B: Bid(ctx, ad)
    P-->>S: []*Bid
    S-->>S: topBids + append SeatBid
  end
  S-->>Caller: BidResponse
```

