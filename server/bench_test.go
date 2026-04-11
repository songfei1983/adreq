package server

import (
	"context"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/songfei1983/adreq/infra"
	"github.com/songfei1983/adreq/internal/benchutil"
	"github.com/songfei1983/adreq/model"
)

func newBenchServer(impCount, candsPerImp, maxImpConcurrent, maxCandConcurrent int) (*AdServer, *model.BidRequest) {
	req := benchutil.MakeRequest("bench_req", impCount, 0)
	source := benchutil.StaticCandidateSource{
		ByImp: benchutil.MakeCandidatesByImp(req.Imp, candsPerImp),
	}
	processor := benchutil.PassthroughProcessor{}
	s := NewAdServer(source, processor, Config{
		RequestTimeout:          10 * time.Second,
		MaxBidsPerImp:           2,
		MaxConcurrentImps:       maxImpConcurrent,
		MaxConcurrentCandidates: maxCandConcurrent,
	})
	return s, req
}

func BenchmarkHandleRequest_Direct(b *testing.B) {
	b.ReportAllocs()

	type tc struct {
		name            string
		imps            int
		candsPerImp     int
		maxImpConc      int
		maxCandConc     int
		parallelismMult int
	}

	cases := []tc{
		{name: "imps=2,cands=10,conc=2x", imps: 2, candsPerImp: 10, maxImpConc: 2, maxCandConc: 2, parallelismMult: 1},
		{name: "imps=2,cands=10,conc=2x,par=4x", imps: 2, candsPerImp: 10, maxImpConc: 2, maxCandConc: 2, parallelismMult: 4},
		{name: "imps=6,cands=40,conc=6x,par=1x", imps: 6, candsPerImp: 40, maxImpConc: 6, maxCandConc: 6, parallelismMult: 1},
		{name: "imps=6,cands=40,conc=6x,par=4x", imps: 6, candsPerImp: 40, maxImpConc: 6, maxCandConc: 6, parallelismMult: 4},
	}

	for i := range cases {
		c := cases[i]
		b.Run(c.name, func(b *testing.B) {
			s, req := newBenchServer(c.imps, c.candsPerImp, c.maxImpConc, c.maxCandConc)
			b.SetParallelism(c.parallelismMult)

			var errs int64
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					resp, err := s.HandleRequest(context.Background(), req)
					if err != nil || resp == nil {
						atomic.AddInt64(&errs, 1)
					}
				}
			})
			b.StopTimer()

			if errs != 0 {
				b.Fatalf("unexpected errors: %d", errs)
			}
		})
	}
}

func BenchmarkHandleRequest_WorkerPool(b *testing.B) {
	b.ReportAllocs()

	type tc struct {
		name            string
		imps            int
		candsPerImp     int
		poolWorkers     int
		poolQueue       int
		parallelismMult int
		expectReject    bool
	}

	procs := runtime.GOMAXPROCS(0)
	cases := []tc{
		{name: "imps=2,cands=10,pool=4p,queue=64k,par=1x", imps: 2, candsPerImp: 10, poolWorkers: 4 * procs, poolQueue: 64 * 1024, parallelismMult: 1, expectReject: false},
		{name: "imps=2,cands=10,pool=4p,queue=64k,par=4x", imps: 2, candsPerImp: 10, poolWorkers: 4 * procs, poolQueue: 64 * 1024, parallelismMult: 4, expectReject: false},
		{name: "imps=6,cands=40,pool=4p,queue=64k,par=1x", imps: 6, candsPerImp: 40, poolWorkers: 4 * procs, poolQueue: 64 * 1024, parallelismMult: 1, expectReject: false},
		{name: "imps=6,cands=40,pool=4p,queue=64k,par=4x", imps: 6, candsPerImp: 40, poolWorkers: 4 * procs, poolQueue: 64 * 1024, parallelismMult: 4, expectReject: false},
		{name: "imps=6,cands=40,pool=1p,queue=0,par=16x", imps: 6, candsPerImp: 40, poolWorkers: procs, poolQueue: 0, parallelismMult: 16, expectReject: true},
	}

	for i := range cases {
		c := cases[i]
		b.Run(c.name, func(b *testing.B) {
			s, req := newBenchServer(c.imps, c.candsPerImp, c.imps, c.imps)
			pool := infra.NewWorkerPool(c.poolWorkers, c.poolQueue)
			b.Cleanup(func() {
				pool.Close()
				s.Shutdown()
			})
			s = s.WithWorkerPool(pool)
			b.SetParallelism(c.parallelismMult)

			var errs int64
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					resp, err := s.HandleRequestWithPool(context.Background(), req)
					if err != nil || resp == nil {
						atomic.AddInt64(&errs, 1)
					}
				}
			})
			b.StopTimer()

			if b.N > 0 {
				b.ReportMetric(100*float64(errs)/float64(b.N), "%rejected")
			}
			if !c.expectReject && errs != 0 {
				b.Fatalf("unexpected rejections: %d", errs)
			}
		})
	}
}

func BenchmarkHandleRequest_WorkerPoolQueueSweep(b *testing.B) {
	b.ReportAllocs()

	procs := runtime.GOMAXPROCS(0)
	s, req := newBenchServer(6, 40, 6, 6)

	for _, queue := range []int{0, 64, 1024, 64 * 1024} {
		queue := queue
		b.Run("queue="+strconv.Itoa(queue), func(b *testing.B) {
			pool := infra.NewWorkerPool(4*procs, queue)
			b.Cleanup(func() {
				pool.Close()
			})
			ss := s.WithWorkerPool(pool)

			var errs int64
			b.SetParallelism(4)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					resp, err := ss.HandleRequestWithPool(context.Background(), req)
					if err != nil || resp == nil {
						atomic.AddInt64(&errs, 1)
					}
				}
			})
			b.StopTimer()

			if b.N > 0 {
				b.ReportMetric(100*float64(errs)/float64(b.N), "%rejected")
			}
		})
	}
}
