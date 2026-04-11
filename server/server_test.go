package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/songfei1983/adreq/bidder"
	"github.com/songfei1983/adreq/executor"
	"github.com/songfei1983/adreq/filter"
	"github.com/songfei1983/adreq/infra"
	"github.com/songfei1983/adreq/model"
)

func newTestServer(maxConcurrent int) *AdServer {
	budget := infra.NewBudgetStore()
	budget.Set("camp_0", 1000)
	budget.Set("camp_1", 2000)
	budget.Set("camp_2", 1500)

	filters := []filter.Filter{
		filter.NewFraudChecker(),
		filter.NewSizeFilter(),
		filter.NewFloorFilter(),
		filter.NewTargetingFilter(),
		filter.NewBudgetFilter(budget),
		filter.NewFrequencyFilter(),
	}

	fc := filter.NewChain(filters)
	defaultBidder := infra.NewDefaultBidder()
	processor := bidder.NewProcessor(defaultBidder, fc)
	source := infra.NewRandomCandidateSource(0)
	return NewAdServer(source, processor, Config{
		MaxConcurrentImps:       maxConcurrent,
		MaxConcurrentCandidates: maxConcurrent,
		Budget:                  budget,
	})
}

func TestHandleRequest(t *testing.T) {
	server := newTestServer(10)

	req := &model.BidRequest{
		ID: "test_req_1",
		Imp: []model.Imp{
			{ID: "imp_1", BidFloor: 1.0, Banner: &model.Banner{W: 300, H: 250}},
			{ID: "imp_2", BidFloor: 2.0, Banner: &model.Banner{W: 728, H: 90}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resp, err := server.HandleRequest(ctx, req)
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.ID != req.ID {
		t.Errorf("Response ID mismatch: got %s, want %s", resp.ID, req.ID)
	}
	for _, sb := range resp.SeatBid {
		if len(sb.Bid) > 2 {
			t.Fatalf("Expected <=2 bids per seatbid, got %d", len(sb.Bid))
		}
	}

	t.Logf("Response: %+v", resp)
}

func TestHandleRequestWithWorkerPool(t *testing.T) {
	pool := infra.NewWorkerPool(5, 100)
	defer pool.Close()

	server := newTestServer(10)
	server = server.WithWorkerPool(pool)

	req := &model.BidRequest{
		ID: "test_req_pool",
		Imp: []model.Imp{
			{ID: "imp_pool_1", BidFloor: 1.0},
			{ID: "imp_pool_2", BidFloor: 1.5},
			{ID: "imp_pool_3", BidFloor: 2.0},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resp, err := server.HandleRequestWithPool(ctx, req)
	if err != nil {
		t.Fatalf("HandleRequestWithPool failed: %v", err)
	}

	if resp.ID != req.ID {
		t.Errorf("Response ID mismatch: got %s, want %s", resp.ID, req.ID)
	}
	if len(resp.SeatBid) == 0 {
		t.Fatalf("Expected at least one seatbid, got 0")
	}
	for _, sb := range resp.SeatBid {
		if len(sb.Bid) > 2 {
			t.Fatalf("Expected <=2 bids per seatbid, got %d", len(sb.Bid))
		}
	}

	t.Logf("Response with pool: %+v", resp)
}

func TestConcurrentRequests(t *testing.T) {
	server := newTestServer(20)

	req := &model.BidRequest{
		ID: "concurrent_req",
		Imp: []model.Imp{
			{ID: "imp_c1", BidFloor: 1.0},
			{ID: "imp_c2", BidFloor: 2.0},
		},
	}

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			reqCopy := *req
			reqCopy.ID = req.ID + "_" + string(rune('0'+idx))
			_, err := server.HandleRequest(ctx, &reqCopy)
			if err != nil {
				t.Errorf("Request %d failed: %v", idx, err)
			}
		}(i)
	}

	close(start)
	wg.Wait()
}

type oneImpSource struct {
	ad model.CandidateAd
}

func (s oneImpSource) FetchCandidates(ctx context.Context, imp *model.Imp) ([]model.CandidateAd, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if imp == nil {
		return nil, nil
	}
	ad := s.ad
	ad.ImpID = imp.ID
	return []model.CandidateAd{ad}, nil
}

type blockingProcessor struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *blockingProcessor) ProcessCandidate(ctx context.Context, ad model.CandidateAd) (*model.CandidateDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.entered != nil {
		p.once.Do(func() { close(p.entered) })
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
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

func TestMaxConcurrentRequestsBlock(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})

	s := NewAdServer(oneImpSource{ad: model.CandidateAd{ID: "ad1", CampaignID: "c1", Price: 1}}, &blockingProcessor{
		entered: entered,
		release: release,
	}, Config{
		RequestTimeout:          500 * time.Millisecond,
		MaxConcurrentRequests:   1,
		RequestAdmissionMode:    RequestAdmissionBlock,
		MaxConcurrentImps:       1,
		MaxConcurrentCandidates: 1,
	})

	req := &model.BidRequest{ID: "r1", Imp: []model.Imp{{ID: "imp1"}}}

	err1 := make(chan error, 1)
	go func() {
		_, err := s.HandleRequest(context.Background(), req)
		err1 <- err
	}()

	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("first request did not enter processing")
	}

	err2 := make(chan error, 1)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()
	go func() {
		_, err := s.HandleRequest(ctx2, req)
		err2 <- err
	}()

	select {
	case err := <-err2:
		t.Fatalf("second request returned early: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	close(release)

	if err := <-err1; err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if err := <-err2; err != nil {
		t.Fatalf("second request failed: %v", err)
	}
}

type serialExecutor struct {
	ctx   context.Context
	tasks []func(context.Context) error
}

func (e *serialExecutor) Context() context.Context {
	return e.ctx
}

func (e *serialExecutor) Go(task func(ctx context.Context) error) bool {
	if task == nil {
		return false
	}
	if err := e.ctx.Err(); err != nil {
		return false
	}
	e.tasks = append(e.tasks, task)
	return true
}

func (e *serialExecutor) Wait() error {
	for _, task := range e.tasks {
		if err := task(e.ctx); err != nil {
			return err
		}
	}
	return e.ctx.Err()
}

type serialExecutorFactory struct{}

func (serialExecutorFactory) NewGroup(ctx context.Context, _ int) executor.Executor {
	if ctx == nil {
		ctx = context.Background()
	}
	return &serialExecutor{ctx: ctx}
}

func (serialExecutorFactory) NewPool(ctx context.Context, _ executor.Pool) executor.Executor {
	if ctx == nil {
		ctx = context.Background()
	}
	return &serialExecutor{ctx: ctx}
}

type staticCandidateSource struct {
	byImp map[string][]model.CandidateAd
}

func (s staticCandidateSource) FetchCandidates(ctx context.Context, imp *model.Imp) ([]model.CandidateAd, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if imp == nil {
		return nil, nil
	}
	return s.byImp[imp.ID], nil
}

type passthroughProcessor struct{}

func (passthroughProcessor) ProcessCandidate(ctx context.Context, ad model.CandidateAd) (*model.CandidateDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bid := &model.Bid{
		ID:       ad.ID,
		ImpID:    ad.ImpID,
		Price:    ad.Price,
		Currency: "USD",
		Ext: model.BidExt{
			CampaignID: ad.CampaignID,
			Priority:   ad.Priority,
			DealID:     ad.DealID,
		},
	}
	return &model.CandidateDecision{
		Bid: bid,
		Spend: &model.SpendIntent{
			CampaignID: ad.CampaignID,
			Amount:     ad.Price,
		},
	}, nil
}

type rejectCampaignBudget struct {
	reject string
}

func (b rejectCampaignBudget) Deduct(campaignID string, _ float64) bool {
	return campaignID != b.reject
}

func TestBudgetDeductRejectsTopBid(t *testing.T) {
	source := staticCandidateSource{
		byImp: map[string][]model.CandidateAd{
			"imp_1": {
				{ID: "ad_1", ImpID: "imp_1", CampaignID: "reject", Price: 10},
				{ID: "ad_2", ImpID: "imp_1", CampaignID: "ok", Price: 9},
			},
		},
	}
	processor := passthroughProcessor{}
	s := NewAdServer(source, processor, Config{
		MaxBidsPerImp:           1,
		MaxConcurrentImps:       10,
		MaxConcurrentCandidates: 10,
		Budget:                  rejectCampaignBudget{reject: "reject"},
		ExecutorFactory:         serialExecutorFactory{},
	})

	req := &model.BidRequest{
		ID: "req_1",
		Imp: []model.Imp{
			{ID: "imp_1", BidFloor: 0},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resp, err := s.HandleRequest(ctx, req)
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if len(resp.SeatBid) != 1 {
		t.Fatalf("Expected 1 seatbid, got %d", len(resp.SeatBid))
	}
	if len(resp.SeatBid[0].Bid) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(resp.SeatBid[0].Bid))
	}
	if resp.SeatBid[0].Bid[0].Ext.CampaignID != "ok" {
		t.Fatalf("Expected campaign ok, got %s", resp.SeatBid[0].Bid[0].Ext.CampaignID)
	}
}
