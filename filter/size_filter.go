package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type SizeFilter struct {
}

func NewSizeFilter() *SizeFilter {
	return &SizeFilter{}
}

func (s *SizeFilter) Name() string {
	return "size_filter"
}

func (s *SizeFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	for _, attr := range ad.TargetAttrs {
		if attr == ad.Position {
			return true, nil
		}
	}
	return true, nil
}
