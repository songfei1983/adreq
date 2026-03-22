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
	processor      ImpProcessor
	pool           *bidder.WorkerPool
	requestTimeout time.Duration
	maxBidsPerImp  int
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
		processor:      processor,
		requestTimeout: 100 * time.Millisecond,
		maxBidsPerImp:  2,
	}
}

func (s *AdServer) HandleRequest(ctx context.Context, req *model.BidRequest) (*model.BidResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	response := &model.BidResponse{
		ID: req.ID,
	}

	if len(req.Imp) == 0 {
		return response, nil
	}

	var wg sync.WaitGroup
	seatBids := make([][]model.Bid, len(req.Imp))

	for i := range req.Imp {
		imp := &req.Imp[i]
		wg.Add(1)
		go func(idx int, i *model.Imp) {
			defer wg.Done()

			select {
			case <-reqCtx.Done():
				return
			default:
			}

			bids, err := s.processor.ProcessImp(reqCtx, i)
			if err != nil {
				return
			}

			top := topBids(bids, s.maxBidsPerImp)
			seatBids[idx] = toBidValues(top)
		}(i, imp)
	}

	wg.Wait()

	if err := reqCtx.Err(); err != nil {
		return nil, err
	}

	for _, bids := range seatBids {
		if len(bids) == 0 {
			continue
		}
		response.SeatBid = append(response.SeatBid, model.SeatBid{Bid: bids})
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

	reqCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	response := &model.BidResponse{
		ID: req.ID,
	}

	var wg sync.WaitGroup
	seatBids := make([][]model.Bid, len(req.Imp))

	for i := range req.Imp {
		idx := i
		imp := req.Imp[i]
		wg.Add(1)

		job := bidder.Job(func(_ context.Context) {
			defer wg.Done()

			select {
			case <-reqCtx.Done():
				return
			default:
			}

			bids, err := s.processor.ProcessImp(reqCtx, &imp)
			if err != nil {
				return
			}

			top := topBids(bids, s.maxBidsPerImp)
			seatBids[idx] = toBidValues(top)
		})

		if !s.pool.Submit(reqCtx, job) {
			wg.Done()
			return nil, fmt.Errorf("pool full, request rejected")
		}
	}

	wg.Wait()

	if err := reqCtx.Err(); err != nil {
		return nil, err
	}

	for _, bids := range seatBids {
		if len(bids) == 0 {
			continue
		}
		response.SeatBid = append(response.SeatBid, model.SeatBid{Bid: bids})
	}

	return response, nil
}
