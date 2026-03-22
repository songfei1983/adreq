package bidder

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/songfei1983/adreq/filter"
	"github.com/songfei1983/adreq/model"
)

type Bidder interface {
	Bid(ctx context.Context, ad *model.CandidateAd) (*model.Bid, error)
}

type Auction struct {
	mu         sync.Mutex
	candidates []*model.CandidateAd
}

func NewAuction() *Auction {
	return &Auction{}
}

func (a *Auction) AddCandidate(ad *model.CandidateAd) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.candidates = append(a.candidates, ad)
}

func (a *Auction) GetTopBid(limit int) []*model.Bid {
	a.mu.Lock()
	defer a.mu.Unlock()

	sorted := make([]*model.CandidateAd, len(a.candidates))
	copy(sorted, a.candidates)

	time.Sleep(1 * time.Millisecond)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Price > sorted[i].Price {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	if limit > len(sorted) {
		limit = len(sorted)
	}

	bids := make([]*model.Bid, 0, limit)
	for i := 0; i < limit; i++ {
		bids = append(bids, &model.Bid{
			ID:       fmt.Sprintf("bid_%s_%d", sorted[i].ImpID, i),
			ImpID:    sorted[i].ImpID,
			Price:    sorted[i].Price,
			Currency: "USD",
			Ext: model.BidExt{
				Priority:   sorted[i].Priority,
				CampaignID: sorted[i].CampaignID,
				DealID:     sorted[i].DealID,
			},
		})
	}
	return bids
}

type Processor struct {
	bidder      Bidder
	filterChain *filter.Chain
	semaphore   chan struct{}
}

func NewProcessor(bidder Bidder, fc *filter.Chain, maxConcurrent int) *Processor {
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

	resultCh := make(chan []*model.Bid, len(candidates))
	errCh := make(chan error, len(candidates))
	var wg sync.WaitGroup

	for _, candidate := range candidates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		p.semaphore <- struct{}{}
		wg.Add(1)

		go func(ad *model.CandidateAd) {
			defer wg.Done()
			defer func() { <-p.semaphore }()

			passed, err := p.filterChain.Apply(ctx, ad)
			if err != nil {
				errCh <- err
				return
			}
			if !passed {
				return
			}

			bid, err := p.bidder.Bid(ctx, ad)
			if err != nil {
				errCh <- err
				return
			}
			if bid != nil {
				resultCh <- []*model.Bid{bid}
			}
		}(candidate)
	}

	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()

	var bids []*model.Bid
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result, ok := <-resultCh:
			if !ok {
				return bids, nil
			}
			bids = append(bids, result...)
		case err := <-errCh:
			if err != nil {
				continue
			}
		}
	}
}

func (p *Processor) fetchCandidates(ctx context.Context, imp *model.Imp) []*model.CandidateAd {
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(5 * time.Millisecond):
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

type DefaultBidder struct{}

func NewDefaultBidder() *DefaultBidder {
	return &DefaultBidder{}
}

func (b *DefaultBidder) Bid(ctx context.Context, ad *model.CandidateAd) (*model.Bid, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Millisecond):
	}

	return &model.Bid{
		ID:       ad.ID,
		ImpID:    ad.ImpID,
		Price:    ad.Price,
		Currency: "USD",
		Ext: model.BidExt{
			CampaignID: ad.CampaignID,
			Priority:   ad.Priority,
		},
	}, nil
}
