package server

import (
	"context"

	"github.com/songfei1983/adreq/model"
)

type ImpProcessor interface {
	ProcessCandidate(ctx context.Context, ad model.CandidateAd) (*model.CandidateDecision, error)
}

type CandidateSource interface {
	FetchCandidates(ctx context.Context, imp *model.Imp) ([]model.CandidateAd, error)
}
