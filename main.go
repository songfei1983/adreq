package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/songfei1983/adreq/bidder"
	"github.com/songfei1983/adreq/model"
	"github.com/songfei1983/adreq/server"
)

func main() {
	directServer := server.NewAdServer(10)
	runDemo(directServer, "Direct Processing", false)

	pool := bidder.NewWorkerPool(5, 100)
	defer pool.Close()

	poolServer := server.NewAdServer(10)
	poolServer = poolServer.WithWorkerPool(pool)
	runDemo(poolServer, "Worker Pool Processing", true)
}

func runDemo(s *server.AdServer, name string, usePool bool) {
	req := &model.BidRequest{
		ID:        "demo_req",
		Timestamp: time.Now(),
		Imp: []model.Imp{
			{ID: "imp_1", BidFloor: 1.0, Banner: &model.Banner{W: 300, H: 250}},
			{ID: "imp_2", BidFloor: 2.0, Banner: &model.Banner{W: 728, H: 90}},
			{ID: "imp_3", BidFloor: 1.5, Banner: &model.Banner{W: 320, H: 50}},
		},
		Device: &model.Device{
			IDFA: "test_idfa",
			IP:   "192.168.1.1",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var resp *model.BidResponse
	var err error

	start := time.Now()

	if usePool {
		resp, err = s.HandleRequestWithPool(ctx, req)
	} else {
		resp, err = s.HandleRequest(ctx, req)
	}

	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[%s] Error: %v", name, err)
		return
	}

	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("[%s] Elapsed: %v\n%s\n\n", name, elapsed, respJSON)
}
