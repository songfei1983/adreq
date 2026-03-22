package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type Filter interface {
	Name() string
	Filter(ctx context.Context, ad *model.CandidateAd) (bool, error)
}
