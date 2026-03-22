package filter

import (
	"context"
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
