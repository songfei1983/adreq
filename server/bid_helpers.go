package server

import (
	"sort"

	"github.com/songfei1983/adreq/model"
)

func topBids(bids []*model.Bid, limit int) []*model.Bid {
	if limit <= 0 || len(bids) == 0 {
		return nil
	}
	if limit > len(bids) {
		limit = len(bids)
	}

	sorted := make([]*model.Bid, len(bids))
	copy(sorted, bids)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Price > sorted[j].Price
	})
	return sorted[:limit]
}

func toBidValues(bids []*model.Bid) []model.Bid {
	if len(bids) == 0 {
		return nil
	}
	out := make([]model.Bid, len(bids))
	for i, b := range bids {
		out[i] = *b
	}
	return out
}
