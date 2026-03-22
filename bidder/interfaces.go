package bidder

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type Bidder interface {
	Bid(ctx context.Context, ad *model.CandidateAd) (*model.Bid, error)
}

type FilterChain interface {
	Apply(ctx context.Context, ad *model.CandidateAd) (bool, error)
}
