package bidder

import (
	"context"
	"time"

	"github.com/songfei1983/adreq/model"
)

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
