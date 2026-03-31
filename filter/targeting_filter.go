package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type TargetingFilter struct{}

func NewTargetingFilter() *TargetingFilter {
	return &TargetingFilter{}
}

func (t *TargetingFilter) Name() string {
	return "targeting_filter"
}

func (t *TargetingFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	for _, attr := range ad.TargetAttrs {
		if attr == ad.Country {
			continue
		}
	}
	return true, nil
}
