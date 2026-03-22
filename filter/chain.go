package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type Chain struct {
	filters []Filter
}

func NewChain(filters []Filter) *Chain {
	return &Chain{filters: filters}
}

func (c *Chain) Apply(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	for _, f := range c.filters {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			passed, err := f.Filter(ctx, ad)
			if err != nil {
				return false, err
			}
			if !passed {
				return false, nil
			}
		}
	}
	return true, nil
}
