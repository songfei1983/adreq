package server

import (
	"sort"

	"github.com/songfei1983/adreq/model"
)

func addTopBid(bids []*model.Bid, bid *model.Bid, limit int) []*model.Bid {
	if bid == nil || limit <= 0 {
		return bids
	}
	if len(bids) == 0 {
		return []*model.Bid{bid}
	}

	pos := len(bids)
	for i := 0; i < len(bids); i++ {
		if bid.Price > bids[i].Price {
			pos = i
			break
		}
	}

	if len(bids) >= limit && pos >= limit {
		return bids
	}

	bids = append(bids, nil)
	copy(bids[pos+1:], bids[pos:])
	bids[pos] = bid

	if len(bids) > limit {
		bids = bids[:limit]
	}
	return bids
}

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
