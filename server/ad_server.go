package server

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/songfei1983/adreq/executor"
	"github.com/songfei1983/adreq/model"
)

type ExecutorFactory interface {
	NewGroup(ctx context.Context, maxConcurrent int) executor.Executor
	NewPool(ctx context.Context, pool executor.Pool) executor.Executor
}

type defaultExecutorFactory struct{}

func (defaultExecutorFactory) NewGroup(ctx context.Context, maxConcurrent int) executor.Executor {
	return executor.NewGroup(ctx, maxConcurrent)
}

func (defaultExecutorFactory) NewPool(ctx context.Context, pool executor.Pool) executor.Executor {
	return executor.NewPoolExecutor(ctx, pool)
}

type Config struct {
	RequestTimeout          time.Duration
	MaxBidsPerImp           int
	MaxConcurrentRequests   int
	RequestAdmissionMode    RequestAdmissionMode
	MaxConcurrentImps       int
	MaxConcurrentCandidates int
	Budget                  BudgetDeductor
	Finalizer               BidFinalizer
	ExecutorFactory         ExecutorFactory
}

type RequestAdmissionMode int

const (
	RequestAdmissionReject RequestAdmissionMode = iota
	RequestAdmissionBlock
)

type BudgetDeductor interface {
	Deduct(campaignID string, amount float64) bool
}

type BidFinalizer interface {
	Finalize(bidsByIdx [][]*model.CandidateDecision, maxBidsPerImp int) ([]model.SeatBid, error)
}

type defaultBidFinalizer struct {
	budget BudgetDeductor
}

func (f defaultBidFinalizer) Finalize(bidsByIdx [][]*model.CandidateDecision, maxBidsPerImp int) ([]model.SeatBid, error) {
	out := make([]model.SeatBid, 0, len(bidsByIdx))
	for i := range bidsByIdx {
		decisions := bidsByIdx[i]
		if len(decisions) == 0 {
			continue
		}

		sorted := make([]*model.CandidateDecision, 0, len(decisions))
		for _, d := range decisions {
			if d == nil || d.Bid == nil {
				continue
			}
			sorted = append(sorted, d)
		}
		if len(sorted) == 0 {
			continue
		}

		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Bid.Price > sorted[j].Bid.Price
		})

		selected := make([]model.Bid, 0, maxBidsPerImp)
		for _, d := range sorted {
			if maxBidsPerImp > 0 && len(selected) >= maxBidsPerImp {
				break
			}
			if f.budget != nil && d.Spend != nil {
				if !f.budget.Deduct(d.Spend.CampaignID, d.Spend.Amount) {
					continue
				}
			}
			selected = append(selected, *d.Bid)
		}

		if len(selected) == 0 {
			continue
		}
		out = append(out, model.SeatBid{Bid: selected})
	}
	return out, nil
}

type AdServer struct {
	source            CandidateSource
	processor         ImpProcessor
	pool              executor.Pool
	execFactory       ExecutorFactory
	requestTimeout    time.Duration
	maxBidsPerImp     int
	reqSem            chan struct{}
	reqAdmission      RequestAdmissionMode
	maxConcurrent     int
	maxCandConcurrent int
	finalizer         BidFinalizer
}

var ErrOverloaded = errors.New("server: overloaded")

func (s *AdServer) acquireRequest(ctx context.Context) error {
	if s.reqSem == nil {
		return nil
	}
	switch s.reqAdmission {
	case RequestAdmissionBlock:
		select {
		case s.reqSem <- struct{}{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	default:
		select {
		case s.reqSem <- struct{}{}:
			return nil
		default:
			return ErrOverloaded
		}
	}
}

func (s *AdServer) releaseRequest() {
	if s.reqSem != nil {
		<-s.reqSem
	}
}

func NewAdServer(source CandidateSource, processor ImpProcessor, cfg Config) *AdServer {
	if source == nil {
		panic("nil source")
	}
	if processor == nil {
		panic("nil processor")
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 100 * time.Millisecond
	}
	if cfg.MaxBidsPerImp <= 0 {
		cfg.MaxBidsPerImp = 2
	}
	if cfg.ExecutorFactory == nil {
		cfg.ExecutorFactory = defaultExecutorFactory{}
	}
	if cfg.Finalizer == nil {
		cfg.Finalizer = defaultBidFinalizer{budget: cfg.Budget}
	}
	var reqSem chan struct{}
	if cfg.MaxConcurrentRequests > 0 {
		reqSem = make(chan struct{}, cfg.MaxConcurrentRequests)
	}
	return &AdServer{
		source:            source,
		processor:         processor,
		execFactory:       cfg.ExecutorFactory,
		requestTimeout:    cfg.RequestTimeout,
		maxBidsPerImp:     cfg.MaxBidsPerImp,
		reqSem:            reqSem,
		reqAdmission:      cfg.RequestAdmissionMode,
		maxConcurrent:     cfg.MaxConcurrentImps,
		maxCandConcurrent: cfg.MaxConcurrentCandidates,
		finalizer:         cfg.Finalizer,
	}
}

func (s *AdServer) HandleRequest(ctx context.Context, req *model.BidRequest) (*model.BidResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	if err := s.acquireRequest(reqCtx); err != nil {
		return nil, err
	}
	defer s.releaseRequest()

	response := &model.BidResponse{
		ID: req.ID,
	}

	if len(req.Imp) == 0 {
		return response, nil
	}

	maxConcurrent := s.maxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = len(req.Imp)
	}

	fetchExec := s.execFactory.NewGroup(reqCtx, maxConcurrent)
	candidatesByIdx := make([][]model.CandidateAd, len(req.Imp))
	for i := range req.Imp {
		idx := i
		imp := req.Imp[i]
		fetchExec.Go(func(ctx context.Context) error {
			candidates, err := s.source.FetchCandidates(ctx, &imp)
			if err != nil {
				return err
			}
			candidatesByIdx[idx] = candidates
			return nil
		})
	}

	if err := fetchExec.Wait(); err != nil {
		return nil, err
	}

	maxCandConcurrent := s.maxCandConcurrent
	if maxCandConcurrent <= 0 {
		maxCandConcurrent = maxConcurrent
	}

	candExec := s.execFactory.NewGroup(reqCtx, maxCandConcurrent)
	decisionsByIdx := make([][]*model.CandidateDecision, len(req.Imp))
	decisionsMu := make([]sync.Mutex, len(req.Imp))

	for i := range candidatesByIdx {
		impIdx := i
		for _, candidate := range candidatesByIdx[impIdx] {
			ad := candidate
			candExec.Go(func(ctx context.Context) error {
				decision, err := s.processor.ProcessCandidate(ctx, ad)
				if err != nil {
					return err
				}
				if decision == nil {
					return nil
				}
				decisionsMu[impIdx].Lock()
				decisionsByIdx[impIdx] = append(decisionsByIdx[impIdx], decision)
				decisionsMu[impIdx].Unlock()
				return nil
			})
		}
	}

	if err := candExec.Wait(); err != nil {
		return nil, err
	}

	seatBids, err := s.finalizer.Finalize(decisionsByIdx, s.maxBidsPerImp)
	if err != nil {
		return nil, err
	}
	response.SeatBid = append(response.SeatBid, seatBids...)
	return response, nil
}

func (s *AdServer) Shutdown() {
	if s.pool != nil {
		s.pool.Close()
		s.pool.Wait()
	}
}

func (s *AdServer) WithWorkerPool(pool executor.Pool) *AdServer {
	s.pool = pool
	return s
}

func (s *AdServer) HandleRequestWithPool(ctx context.Context, req *model.BidRequest) (*model.BidResponse, error) {
	if s.pool == nil {
		return s.HandleRequest(ctx, req)
	}

	reqCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	if err := s.acquireRequest(reqCtx); err != nil {
		return nil, err
	}
	defer s.releaseRequest()

	response := &model.BidResponse{
		ID: req.ID,
	}

	if len(req.Imp) == 0 {
		return response, nil
	}

	fetchExec := s.execFactory.NewPool(reqCtx, s.pool)
	candidatesByIdx := make([][]model.CandidateAd, len(req.Imp))
	for i := range req.Imp {
		idx := i
		imp := req.Imp[i]
		fetchExec.Go(func(ctx context.Context) error {
			candidates, err := s.source.FetchCandidates(ctx, &imp)
			if err != nil {
				return err
			}
			candidatesByIdx[idx] = candidates
			return nil
		})
	}

	if err := fetchExec.Wait(); err != nil {
		if errors.Is(err, executor.ErrRejected) {
			return nil, fmt.Errorf("pool full, request rejected")
		}
		return nil, err
	}

	candExec := s.execFactory.NewPool(reqCtx, s.pool)
	decisionsByIdx := make([][]*model.CandidateDecision, len(req.Imp))
	decisionsMu := make([]sync.Mutex, len(req.Imp))

	for i := range candidatesByIdx {
		impIdx := i
		for _, candidate := range candidatesByIdx[impIdx] {
			ad := candidate
			candExec.Go(func(ctx context.Context) error {
				decision, err := s.processor.ProcessCandidate(ctx, ad)
				if err != nil {
					return err
				}
				if decision == nil {
					return nil
				}
				decisionsMu[impIdx].Lock()
				decisionsByIdx[impIdx] = append(decisionsByIdx[impIdx], decision)
				decisionsMu[impIdx].Unlock()
				return nil
			})
		}
	}

	if err := candExec.Wait(); err != nil {
		if errors.Is(err, executor.ErrRejected) {
			return nil, fmt.Errorf("pool full, request rejected")
		}
		return nil, err
	}

	seatBids, err := s.finalizer.Finalize(decisionsByIdx, s.maxBidsPerImp)
	if err != nil {
		return nil, err
	}
	response.SeatBid = append(response.SeatBid, seatBids...)
	return response, nil
}
