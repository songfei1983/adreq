package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/songfei1983/adreq/bidder"
	"github.com/songfei1983/adreq/filter"
	"github.com/songfei1983/adreq/model"
)

type AdServer struct {
	processor *bidder.Processor
	pool      *bidder.WorkerPool
}

func NewAdServer(maxConcurrent int) *AdServer {
	budgetCache := filter.NewBudgetCache()
	budgetCache.Set("camp_0", 1000)
	budgetCache.Set("camp_1", 2000)
	budgetCache.Set("camp_2", 1500)

	filters := []filter.Filter{
		filter.NewFraudChecker(),
		filter.NewSizeFilter(),
		filter.NewFloorFilter(),
		filter.NewTargetingFilter(),
		filter.NewBudgetFilter(budgetCache),
		filter.NewFrequencyFilter(),
	}

	fc := filter.NewChain(filters)
	defaultBidder := bidder.NewDefaultBidder()
	processor := bidder.NewProcessor(defaultBidder, fc, maxConcurrent)

	return &AdServer{
		processor: processor,
	}
}

func (s *AdServer) HandleRequest(ctx context.Context, req *model.BidRequest) (*model.BidResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	response := &model.BidResponse{
		ID: req.ID,
	}

	if len(req.Imp) == 0 {
		return response, nil
	}

	impCh := make(chan *model.Imp, len(req.Imp))
	resultCh := make(chan *bidder.Auction, len(req.Imp))
	errCh := make(chan error, len(req.Imp))
	var wg sync.WaitGroup

	for i := range req.Imp {
		impCh <- &req.Imp[i]
	}
	close(impCh)

	for imp := range impCh {
		wg.Add(1)
		go func(i *model.Imp) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			auction := bidder.NewAuction()
			bids, err := s.processor.ProcessImp(ctx, i)
			if err != nil {
				errCh <- err
				return
			}

			for _, bid := range bids {
				auction.AddCandidate(&model.CandidateAd{
					ID:         bid.ID,
					ImpID:      bid.ImpID,
					Price:      bid.Price,
					Priority:   bid.Ext.Priority,
					CampaignID: bid.Ext.CampaignID,
					DealID:     bid.Ext.DealID,
				})
			}

			resultCh <- auction
		}(imp)
	}

	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()

	for auction := range resultCh {
		bids := auction.GetTopBid(2)
		if len(bids) > 0 {
			seatBid := model.SeatBid{
				Bid: make([]model.Bid, len(bids)),
			}
			for i, b := range bids {
				seatBid.Bid[i] = *b
			}
			response.SeatBid = append(response.SeatBid, seatBid)
		}
	}

	for err := range errCh {
		if err != nil {
			continue
		}
	}

	return response, nil
}

func (s *AdServer) Shutdown() {
	if s.pool != nil {
		s.pool.Close()
		s.pool.Wait()
	}
}

func (s *AdServer) WithWorkerPool(pool *bidder.WorkerPool) *AdServer {
	s.pool = pool
	return s
}

func (s *AdServer) HandleRequestWithPool(ctx context.Context, req *model.BidRequest) (*model.BidResponse, error) {
	if s.pool == nil {
		return s.HandleRequest(ctx, req)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	response := &model.BidResponse{
		ID: req.ID,
	}

	var wg sync.WaitGroup
	resultCh := make(chan *model.BidResponse, len(req.Imp))

	for i := range req.Imp {
		imp := req.Imp[i]
		wg.Add(1)

		job := bidder.Job(func(_ context.Context) {
			defer wg.Done()

			resp, err := s.HandleRequest(reqCtx, &model.BidRequest{
				ID:  req.ID,
				Imp: []model.Imp{imp},
			})
			if err != nil {
				return
			}

			select {
			case resultCh <- resp:
			case <-reqCtx.Done():
			}
		})

		if !s.pool.Submit(job) {
			wg.Done()
			return nil, fmt.Errorf("pool full, request rejected")
		}
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for resp := range resultCh {
		response.SeatBid = append(response.SeatBid, resp.SeatBid...)
	}

	return response, nil
}
