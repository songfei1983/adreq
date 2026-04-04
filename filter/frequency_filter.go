package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type FrequencyFilter struct{}

func NewFrequencyFilter() *FrequencyFilter {
	return &FrequencyFilter{}
}

func (f *FrequencyFilter) Name() string {
	return "frequency_filter"
}

func (f *FrequencyFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	return true, nil
}
