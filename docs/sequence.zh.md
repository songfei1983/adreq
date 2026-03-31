# 时序图

本工程演示两种处理模式：

- Direct：`server.AdServer` 通过 Executor 统一调度并发（受并发度限制）来并行拉取候选与评估候选；业务逻辑按“单任务同步执行”保持纯净
- WorkerPool：编排模型一致，但 Executor 把任务提交到有界的 `WorkerPool`

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
  loop each Imp
    S->>E: Go(拉取候选)
    E->>CS: FetchCandidates(reqCtx, imp)
    CS-->>E: []CandidateAd
    E-->>S: candidates
  end
  loop each CandidateAd
    S->>E: Go(评估候选)
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
  end
  S-->>Caller: BidResponse

## Worker Pool Processing

```mermaid
sequenceDiagram
  autonumber
  participant Caller as main/runDemo
  participant S as server.AdServer
  participant E as executor.Executor
  participant CS as CandidateSource
  participant WP as bidder.WorkerPool
  participant W as worker goroutine
  participant P as ImpProcessor
  participant FC as filter.Chain
  participant B as bidder.Bidder
  participant Fin as BidFinalizer

  Caller->>S: HandleRequestWithPool(ctx, BidRequest)
  loop each Imp
    S->>E: Go(拉取候选)
    E->>WP: Submit(reqCtx, job)
  end
  loop each fetch job
    WP-->>W: dispatch job
    W->>CS: FetchCandidates(reqCtx, imp)
    CS-->>W: []CandidateAd
    W-->>S: candidates
  end
  loop each CandidateAd
    S->>E: Go(评估候选)
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
