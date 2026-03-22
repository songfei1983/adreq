package filter

import (
	"context"
	"sync"

	"github.com/songfei1983/adreq/model"
)

type FrequencyFilter struct {
	counts sync.Map
}

func NewFrequencyFilter() *FrequencyFilter {
	return &FrequencyFilter{}
}

func (f *FrequencyFilter) Name() string {
	return "frequency_filter"
}

func (f *FrequencyFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	return true, nil
}
