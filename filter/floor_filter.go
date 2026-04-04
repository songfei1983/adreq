package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type FloorFilter struct{}

func NewFloorFilter() *FloorFilter {
	return &FloorFilter{}
}

func (f *FloorFilter) Name() string {
	return "floor_filter"
}

func (f *FloorFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	return ad.PassesFloor(), nil
}
