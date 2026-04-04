package bidder

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type Processor struct {
	bidder      Bidder
	filterChain FilterChain
}

func NewProcessor(bidder Bidder, fc FilterChain) *Processor {
	return &Processor{
		bidder:      bidder,
		filterChain: fc,
	}
}

func (p *Processor) ProcessCandidate(ctx context.Context, ad model.CandidateAd) (*model.CandidateDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	passed, err := p.filterChain.Apply(ctx, &ad)
	if err != nil || !passed {
		return nil, nil
	}

	bid, err := p.bidder.Bid(ctx, &ad)
	if err != nil {
		return nil, nil
	}
	if bid == nil {
		return nil, nil
	}
	return &model.CandidateDecision{
		Bid: bid,
		Spend: &model.SpendIntent{
			CampaignID: bid.Ext.CampaignID,
			Amount:     bid.Price,
		},
	}, nil
}
