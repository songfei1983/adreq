package filter

import (
	"context"
	"sync"

	"github.com/songfei1983/adreq/model"
)

type Filter interface {
	Name() string
	Filter(ctx context.Context, ad *model.CandidateAd) (bool, error)
}

type Chain struct {
	filters []Filter
}

func NewChain(filters []Filter) *Chain {
	return &Chain{filters: filters}
}

func (c *Chain) Apply(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	for _, f := range c.filters {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			passed, err := f.Filter(ctx, ad)
			if err != nil {
				return false, err
			}
			if !passed {
				return false, nil
			}
		}
	}
	return true, nil
}

type BudgetCache struct {
	mu      sync.RWMutex
	budgets map[string]float64
}

func NewBudgetCache() *BudgetCache {
	return &BudgetCache{
		budgets: make(map[string]float64),
	}
}

func (b *BudgetCache) Get(campaignID string) (float64, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	val, ok := b.budgets[campaignID]
	return val, ok
}

func (b *BudgetCache) Set(campaignID string, budget float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.budgets[campaignID] = budget
}

func (b *BudgetCache) Deduct(campaignID string, amount float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	current, ok := b.budgets[campaignID]
	if !ok || current < amount {
		return false
	}
	b.budgets[campaignID] = current - amount
	return true
}
