package bidder

import (
	"fmt"
	"sort"
	"sync"

	"github.com/songfei1983/adreq/model"
)

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

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CalculateEffectivePrice() > sorted[j].CalculateEffectivePrice()
	})

	if limit > len(sorted) {
		limit = len(sorted)
	}

	bids := make([]*model.Bid, 0, limit)
	for i := 0; i < limit; i++ {
		cand := sorted[i]
		bids = append(bids, &model.Bid{
			ID:       fmt.Sprintf("bid_%s_%d", cand.ImpID, i),
			ImpID:    cand.ImpID,
			Price:    cand.CalculateEffectivePrice(),
			Currency: "USD",
			Ext: model.BidExt{
				Priority:   cand.Priority,
				CampaignID: cand.CampaignID,
				DealID:     cand.DealID,
			},
		})
	}
	return bids
}
