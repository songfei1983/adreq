package bidder

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/songfei1983/adreq/model"
)

type Processor struct {
	bidder      Bidder
	filterChain FilterChain
	semaphore   chan struct{}
}

func NewProcessor(bidder Bidder, fc FilterChain, maxConcurrent int) *Processor {
	return &Processor{
		bidder:      bidder,
		filterChain: fc,
		semaphore:   make(chan struct{}, maxConcurrent),
	}
}

func (p *Processor) ProcessImp(ctx context.Context, imp *model.Imp) ([]*model.Bid, error) {
	candidates := p.fetchCandidates(ctx, imp)
	if len(candidates) == 0 {
		return nil, nil
	}

	var wg sync.WaitGroup
	var bidsMu sync.Mutex
	bids := make([]*model.Bid, 0)

	for _, candidate := range candidates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		p.semaphore <- struct{}{}
		wg.Add(1)

		ad := *candidate
		go func() {
			defer wg.Done()
			defer func() { <-p.semaphore }()

			select {
			case <-ctx.Done():
				return
			default:
			}

			passed, err := p.filterChain.Apply(ctx, &ad)
			if err != nil {
				return
			}
			if !passed {
				return
			}

			bid, err := p.bidder.Bid(ctx, &ad)
			if err != nil {
				return
			}
			if bid != nil {
				bidsMu.Lock()
				bids = append(bids, bid)
				bidsMu.Unlock()
			}
		}()
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return bids, nil
}

func (p *Processor) fetchCandidates(ctx context.Context, imp *model.Imp) []*model.CandidateAd {
	timer := time.NewTimer(5 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil
	case <-timer.C:
	}

	candidates := make([]*model.CandidateAd, 0)
	numCandidates := rand.Intn(5) + 1

	for i := 0; i < numCandidates; i++ {
		candidates = append(candidates, &model.CandidateAd{
			ID:          fmt.Sprintf("ad_%d", i),
			ImpID:       imp.ID,
			CampaignID:  fmt.Sprintf("camp_%d", i),
			Price:       float64(rand.Intn(100)) + imp.BidFloor,
			BidFloor:    imp.BidFloor,
			Priority:    rand.Intn(10),
			TargetAttrs: []string{"desktop", "mobile", "US", "UK"},
			Country:     "US",
			Position:    "above",
		})
	}

	return candidates
}
