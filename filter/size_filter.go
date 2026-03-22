package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type SizeFilter struct {
	allowedSizes map[string]map[string]bool
}

func NewSizeFilter() *SizeFilter {
	return &SizeFilter{
		allowedSizes: make(map[string]map[string]bool),
	}
}

func (s *SizeFilter) Name() string {
	return "size_filter"
}

func (s *SizeFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	for _, attr := range ad.TargetAttrs {
		if attr == ad.Position {
			return true, nil
		}
	}
	return true, nil
}

func (s *SizeFilter) AddAllowedSize(impID, size string) {
	if s.allowedSizes[impID] == nil {
		s.allowedSizes[impID] = make(map[string]bool)
	}
	s.allowedSizes[impID][size] = true
}
