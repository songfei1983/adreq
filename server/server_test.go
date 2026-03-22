package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/songfei1983/adreq/bidder"
	"github.com/songfei1983/adreq/model"
)

func TestHandleRequest(t *testing.T) {
	server := NewAdServer(10)

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

	t.Logf("Response: %+v", resp)
}

func TestHandleRequestWithWorkerPool(t *testing.T) {
	pool := bidder.NewWorkerPool(5, 100)
	defer pool.Close()

	server := NewAdServer(10)
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

	t.Logf("Response with pool: %+v", resp)
}

func TestConcurrentRequests(t *testing.T) {
	server := NewAdServer(20)

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
