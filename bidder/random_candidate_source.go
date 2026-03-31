package bidder

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/songfei1983/adreq/model"
)

type RandomCandidateSource struct {
	delay time.Duration
}

func NewRandomCandidateSource(delay time.Duration) *RandomCandidateSource {
	return &RandomCandidateSource{delay: delay}
}

func (s *RandomCandidateSource) FetchCandidates(ctx context.Context, imp *model.Imp) ([]model.CandidateAd, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	numCandidates := rand.Intn(5) + 1
	candidates := make([]model.CandidateAd, 0, numCandidates)
	for i := 0; i < numCandidates; i++ {
		candidates = append(candidates, model.CandidateAd{
			ID:          fmt.Sprintf("ad_%d", i),
			ImpID:       imp.ID,
			CampaignID:  fmt.Sprintf("camp_%d", i),
			Price:       float64(rand.Intn(100)) + imp.BidFloor,
			BidFloor:    imp.BidFloor,
			Priority:    rand.Intn(10),
			TargetAttrs: []string{"desktop", "mobile", "US", "UK"},
			Country:     "US",
			Position:    "above",
		})
	}
	return candidates, nil
}
