package filter

import (
	"context"
	"errors"

	"github.com/songfei1983/adreq/model"
)

type BudgetFilter struct {
	cache *BudgetCache
}

func NewBudgetFilter(cache *BudgetCache) *BudgetFilter {
	return &BudgetFilter{cache: cache}
}

func (b *BudgetFilter) Name() string {
	return "budget_filter"
}

func (b *BudgetFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	if !b.cache.Deduct(ad.CampaignID, ad.Price) {
		return false, errors.New("budget exhausted")
	}
	return true, nil
}
