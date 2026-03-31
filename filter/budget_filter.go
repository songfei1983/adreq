package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type BudgetFilter struct {
	budget BudgetReader
}

type BudgetReader interface {
	Get(campaignID string) (float64, bool)
}

func NewBudgetFilter(budget BudgetReader) *BudgetFilter {
	return &BudgetFilter{budget: budget}
}

func (b *BudgetFilter) Name() string {
	return "budget_filter"
}

func (b *BudgetFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	if b.budget == nil {
		return true, nil
	}
	current, ok := b.budget.Get(ad.CampaignID)
	if !ok || current < ad.Price {
		return false, nil
	}
	return true, nil
}
