# 时序图

本工程演示两种处理模式：

- Direct：请求内对每个 `Imp` 起 goroutine；imp 内部对候选并发（受 `semaphore` 限制）
- WorkerPool：将每个 `Imp` 封装为 job 提交到 `WorkerPool`，由固定 worker 执行；job 内部同样会触发候选并发

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

