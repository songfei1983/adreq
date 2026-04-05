package benchutil

import (
	"context"
	"strconv"

	"github.com/songfei1983/adreq/model"
)

func MakeRequest(id string, impCount int, bidFloor float64) *model.BidRequest {
	if id == "" {
		id = "bench_req"
	}
	if impCount < 0 {
		impCount = 0
	}
	req := &model.BidRequest{
		ID:  id,
		Imp: make([]model.Imp, 0, impCount),
	}
	for i := 0; i < impCount; i++ {
		req.Imp = append(req.Imp, model.Imp{
			ID:       "imp_" + strconv.Itoa(i),
			BidFloor: bidFloor,
		})
	}
	return req
}

func MakeCandidatesByImp(imps []model.Imp, candsPerImp int) map[string][]model.CandidateAd {
	if candsPerImp < 0 {
		candsPerImp = 0
	}
	out := make(map[string][]model.CandidateAd, len(imps))
	for i := range imps {
		impID := imps[i].ID
		cands := make([]model.CandidateAd, 0, candsPerImp)
		for j := 0; j < candsPerImp; j++ {
			price := 1.0 + float64((j%10)+1)
			cands = append(cands, model.CandidateAd{
				ID:         "ad_" + strconv.Itoa(i) + "_" + strconv.Itoa(j),
				ImpID:      impID,
				CampaignID: "camp_" + strconv.Itoa(j%8),
				Price:      price,
				BidFloor:   imps[i].BidFloor,
				Priority:   j % 4,
				DealID:     "",
			})
		}
		out[impID] = cands
	}
	return out
}

type StaticCandidateSource struct {
	ByImp map[string][]model.CandidateAd
}

func (s StaticCandidateSource) FetchCandidates(ctx context.Context, imp *model.Imp) ([]model.CandidateAd, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if imp == nil {
		return nil, nil
	}
	return s.ByImp[imp.ID], nil
}

type PassthroughProcessor struct{}

func (PassthroughProcessor) ProcessCandidate(ctx context.Context, ad model.CandidateAd) (*model.CandidateDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &model.CandidateDecision{
		Bid: &model.Bid{
			ID:       ad.ID,
			ImpID:    ad.ImpID,
			Price:    ad.Price,
			Currency: "USD",
			Ext: model.BidExt{
				CampaignID: ad.CampaignID,
				Priority:   ad.Priority,
				DealID:     ad.DealID,
			},
		},
	}, nil
}
