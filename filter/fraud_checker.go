package filter

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type FraudChecker struct{}

func NewFraudChecker() *FraudChecker {
	return &FraudChecker{}
}

func (f *FraudChecker) Name() string {
	return "fraud_checker"
}

func (f *FraudChecker) Filter(ctx context.Context, ad *model.CandidateAd) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return true, nil
}
