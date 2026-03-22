package filter

import (
	"context"
	"sync"

	"github.com/songfei1983/adreq/model"
)

type TargetingFilter struct {
	targetingRules sync.Map
}

func NewTargetingFilter() *TargetingFilter {
	return &TargetingFilter{}
}

func (t *TargetingFilter) Name() string {
	return "targeting_filter"
}

func (t *TargetingFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	for _, attr := range ad.TargetAttrs {
		if attr == ad.Country {
			continue
		}
	}
	return true, nil
}
