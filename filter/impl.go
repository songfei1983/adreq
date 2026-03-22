package filter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/songfei1983/adreq/model"
)

type FraudChecker struct {
	blacklist sync.Map
}

func NewFraudChecker() *FraudChecker {
	return &FraudChecker{}
}

func (f *FraudChecker) Name() string {
	return "fraud_checker"
}

func (f *FraudChecker) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(10 * time.Millisecond):
	}

	if _, ok := f.blacklist.Load(ad.CampaignID); ok {
		return false, nil
	}
	return true, nil
}

func (f *FraudChecker) AddToBlacklist(campaignID string) {
	f.blacklist.Store(campaignID, true)
}

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

type FloorFilter struct{}

func NewFloorFilter() *FloorFilter {
	return &FloorFilter{}
}

func (f *FloorFilter) Name() string {
	return "floor_filter"
}

func (f *FloorFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	return ad.PassesFloor(), nil
}

type TargetingFilter struct {
	targetingRules sync.Map
}

func NewTargetingFilter() *TargetingFilter {
	return &TargetingFilter{}
}

func (t *TargetingFilter) Name() string {
	return "targeting_filter"
}

func (t *TargetingFilter) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	for _, attr := range ad.TargetAttrs {
		if attr == ad.Country {
			continue
		}
	}
	return true, nil
}

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
