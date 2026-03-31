package infra

import "sync"

type BudgetStore struct {
	mu      sync.RWMutex
	budgets map[string]float64
}

func NewBudgetStore() *BudgetStore {
	return &BudgetStore{
		budgets: make(map[string]float64),
	}
}

func (b *BudgetStore) Get(campaignID string) (float64, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	val, ok := b.budgets[campaignID]
	return val, ok
}

func (b *BudgetStore) Set(campaignID string, budget float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.budgets[campaignID] = budget
}

func (b *BudgetStore) Deduct(campaignID string, amount float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	current, ok := b.budgets[campaignID]
	if !ok || current < amount {
		return false
	}
	b.budgets[campaignID] = current - amount
	return true
}
