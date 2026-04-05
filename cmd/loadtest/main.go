package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/songfei1983/adreq/infra"
	"github.com/songfei1983/adreq/internal/benchutil"
	"github.com/songfei1983/adreq/model"
	"github.com/songfei1983/adreq/server"
)

var cpuSink uint64

func percentile(sorted []int64, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return time.Duration(sorted[0])
	}
	if p >= 1 {
		return time.Duration(sorted[len(sorted)-1])
	}
	i := int(float64(len(sorted)-1) * p)
	return time.Duration(sorted[i])
}

func parseHumanInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	mult := 1
	last := s[len(s)-1]
	if last == 'k' || last == 'K' {
		mult = 1024
		s = s[:len(s)-1]
	} else if last == 'm' || last == 'M' {
		mult = 1024 * 1024
		s = s[:len(s)-1]
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	return n * mult, nil
}

func parseIntList(list string) ([]int, error) {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil, nil
	}
	parts := strings.Split(list, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := parseHumanInt(p)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", p, err)
		}
		out = append(out, n)
	}
	return out, nil
}

type runCfg struct {
	mode        string
	duration    time.Duration
	concurrency int
	imps        int
	candsPerImp int
	maxImpConc  int
	maxCandConc int
	maxRequests int
	poolWorkers int
	poolQueue   int
	timeout     time.Duration
	samples     int
	workIters   int
	workUs      int
}

type runResult struct {
	mode        string
	concurrency int
	poolWorkers int
	poolQueue   int

	elapsed         time.Duration
	total           int64
	ok              int64
	errors          int64
	attemptedPerSec float64
	okPerSec        float64
	successRate     float64

	p50 time.Duration
	p95 time.Duration
	p99 time.Duration

	latencySamples int
}

type workProcessor struct {
	sleep time.Duration
	iters int
}

func (p workProcessor) ProcessCandidate(ctx context.Context, ad model.CandidateAd) (*model.CandidateDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.sleep > 0 {
		time.Sleep(p.sleep)
	}
	if p.iters > 0 {
		x := uint64(1469598103934665603)
		for i := 0; i < p.iters; i++ {
			x ^= uint64(i) + 0x9e3779b97f4a7c15
			x *= 1099511628211
		}
		atomic.AddUint64(&cpuSink, x)
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

func runOnce(cfg runCfg, req *model.BidRequest, candidates map[string][]model.CandidateAd) (runResult, error) {
	source := benchutil.StaticCandidateSource{ByImp: candidates}
	var processor server.ImpProcessor = benchutil.PassthroughProcessor{}
	if cfg.workIters > 0 || cfg.workUs > 0 {
		sleep := time.Duration(cfg.workUs) * time.Microsecond
		processor = workProcessor{sleep: sleep, iters: cfg.workIters}
	}
	s := server.NewAdServer(source, processor, server.Config{
		RequestTimeout:          cfg.timeout,
		MaxBidsPerImp:           2,
		MaxConcurrentRequests:   cfg.maxRequests,
		MaxConcurrentImps:       cfg.maxImpConc,
		MaxConcurrentCandidates: cfg.maxCandConc,
	})
	if cfg.mode == "pool" {
		pool := infra.NewWorkerPool(cfg.poolWorkers, cfg.poolQueue)
		s = s.WithWorkerPool(pool)
		defer s.Shutdown()
	}

	samples := make([]int64, cfg.samples)
	var sampleIdx uint64

	var total int64
	var ok int64
	var errs int64

	start := time.Now()
	deadline := start.Add(cfg.duration)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(cfg.concurrency)
	for i := 0; i < cfg.concurrency; i++ {
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				t0 := time.Now()
				var err error
				if cfg.mode == "pool" {
					_, err = s.HandleRequestWithPool(ctx, req)
				} else {
					_, err = s.HandleRequest(ctx, req)
				}
				dur := time.Since(t0)

				atomic.AddInt64(&total, 1)
				if err == nil {
					atomic.AddInt64(&ok, 1)
					idx := atomic.AddUint64(&sampleIdx, 1) - 1
					if idx < uint64(len(samples)) {
						samples[idx] = dur.Nanoseconds()
					}
				} else {
					atomic.AddInt64(&errs, 1)
				}
			}
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	t := atomic.LoadInt64(&total)
	okCount := atomic.LoadInt64(&ok)
	e := atomic.LoadInt64(&errs)

	sampleN := int(atomic.LoadUint64(&sampleIdx))
	if sampleN > len(samples) {
		sampleN = len(samples)
	}
	sorted := samples[:sampleN]
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	attemptedPerSec := 0.0
	okPerSec := 0.0
	if elapsed > 0 {
		attemptedPerSec = float64(t) / elapsed.Seconds()
		okPerSec = float64(okCount) / elapsed.Seconds()
	}

	successRate := 0.0
	if t > 0 {
		successRate = float64(okCount) / float64(t)
	}

	res := runResult{
		mode:            cfg.mode,
		concurrency:     cfg.concurrency,
		poolWorkers:     cfg.poolWorkers,
		poolQueue:       cfg.poolQueue,
		elapsed:         elapsed,
		total:           t,
		ok:              okCount,
		errors:          e,
		attemptedPerSec: attemptedPerSec,
		okPerSec:        okPerSec,
		successRate:     successRate,
		p50:             percentile(sorted, 0.50),
		p95:             percentile(sorted, 0.95),
		p99:             percentile(sorted, 0.99),
		latencySamples:  sampleN,
	}
	if cfg.mode != "pool" {
		res.poolWorkers = 0
		res.poolQueue = 0
	}
	return res, nil
}

func main() {
	procs := runtime.GOMAXPROCS(0)

	var (
		mode          = flag.String("mode", "direct", "direct|pool")
		sweep         = flag.Bool("sweep", false, "run sweep and print CSV + series summaries")
		duration      = flag.Duration("duration", 10*time.Second, "test duration")
		concurrency   = flag.Int("concurrency", 4*procs, "number of concurrent request loops")
		concurrencyL  = flag.String("concurrency-list", "", "comma list for sweep, e.g. 32,64,128")
		imps          = flag.Int("imps", 6, "imps per request")
		candsPerImp   = flag.Int("cands", 40, "candidates per imp")
		maxRequests   = flag.Int("max-requests", 0, "max concurrent in-flight requests (0 disables)")
		maxImpConc    = flag.Int("max-imp-conc", 6, "direct-mode max concurrent imps (per request)")
		maxCandConc   = flag.Int("max-cand-conc", 6, "direct-mode max concurrent candidates (per request)")
		poolWorkers   = flag.Int("pool-workers", 4*procs, "pool workers")
		poolWorkersL  = flag.String("pool-workers-list", "", "comma list for sweep, e.g. 16,32")
		poolQueue     = flag.Int("pool-queue", 64*1024, "pool queue size")
		poolQueueL    = flag.String("pool-queue-list", "", "comma list for sweep, e.g. 0,64,1024,64k")
		timeout       = flag.Duration("timeout", 10*time.Second, "request timeout")
		sampleLatency = flag.Int("samples", 20000, "number of latency samples to keep")
		workIters     = flag.Int("work-iters", 0, "cpu work iterations per candidate (0 disables)")
		workUs        = flag.Int("work-us", 0, "sleep microseconds per candidate (0 disables)")
	)
	flag.Parse()

	if *mode != "direct" && *mode != "pool" {
		fmt.Fprintf(os.Stderr, "invalid -mode=%s (want direct|pool)\n", *mode)
		os.Exit(2)
	}
	if *concurrency < 1 {
		*concurrency = 1
	}
	if *imps < 0 {
		*imps = 0
	}
	if *candsPerImp < 0 {
		*candsPerImp = 0
	}
	if *maxImpConc < 1 {
		*maxImpConc = *imps
	}
	if *maxCandConc < 1 {
		*maxCandConc = *maxImpConc
	}
	if *sampleLatency < 0 {
		*sampleLatency = 0
	}

	req := benchutil.MakeRequest("load_req", *imps, 0)
	candidates := benchutil.MakeCandidatesByImp(req.Imp, *candsPerImp)

	base := runCfg{
		mode:        *mode,
		duration:    *duration,
		concurrency: *concurrency,
		imps:        *imps,
		candsPerImp: *candsPerImp,
		maxImpConc:  *maxImpConc,
		maxCandConc: *maxCandConc,
		maxRequests: *maxRequests,
		poolWorkers: *poolWorkers,
		poolQueue:   *poolQueue,
		timeout:     *timeout,
		samples:     *sampleLatency,
		workIters:   *workIters,
		workUs:      *workUs,
	}

	if !*sweep {
		res, err := runOnce(base, req, candidates)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Printf("mode=%s duration=%s concurrency=%d imps=%d cands=%d\n", res.mode, res.elapsed.Round(time.Millisecond), res.concurrency, *imps, *candsPerImp)
		fmt.Printf("attempted=%.0f req/s ok=%.0f req/s total=%d ok=%d errors=%d success_rate=%.2f%%\n", res.attemptedPerSec, res.okPerSec, res.total, res.ok, res.errors, 100*res.successRate)
		if res.latencySamples > 0 {
			fmt.Printf("ok_latency p50=%s p95=%s p99=%s samples=%d\n", res.p50, res.p95, res.p99, res.latencySamples)
		}
		return
	}

	concList, err := parseIntList(*concurrencyL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	workersList, err := parseIntList(*poolWorkersL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	queueList, err := parseIntList(*poolQueueL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	if len(concList) == 0 {
		concList = []int{base.concurrency}
	}
	if len(workersList) == 0 {
		workersList = []int{base.poolWorkers}
	}
	if len(queueList) == 0 {
		queueList = []int{base.poolQueue}
	}

	results := make([]runResult, 0, len(concList)*len(workersList)*len(queueList))
	fmt.Println("mode,concurrency,pool_workers,pool_queue,attempted_rps,ok_rps,success_rate,p50_ms,p95_ms,p99_ms,latency_samples,elapsed_ms")
	for _, workers := range workersList {
		for _, queue := range queueList {
			for _, conc := range concList {
				cfg := base
				cfg.concurrency = conc
				cfg.poolWorkers = workers
				cfg.poolQueue = queue
				res, err := runOnce(cfg, req, candidates)
				if err != nil {
					fmt.Fprintln(os.Stderr, err.Error())
					os.Exit(1)
				}
				results = append(results, res)
				fmt.Printf("%s,%d,%d,%d,%.0f,%.0f,%.6f,%.3f,%.3f,%.3f,%d,%d\n",
					res.mode,
					res.concurrency,
					res.poolWorkers,
					res.poolQueue,
					res.attemptedPerSec,
					res.okPerSec,
					res.successRate,
					float64(res.p50)/float64(time.Millisecond),
					float64(res.p95)/float64(time.Millisecond),
					float64(res.p99)/float64(time.Millisecond),
					res.latencySamples,
					res.elapsed.Milliseconds(),
				)
			}
		}
	}

	type seriesKey struct {
		mode        string
		poolWorkers int
		poolQueue   int
	}
	bySeries := make(map[seriesKey][]runResult)
	for _, r := range results {
		k := seriesKey{mode: r.mode, poolWorkers: r.poolWorkers, poolQueue: r.poolQueue}
		bySeries[k] = append(bySeries[k], r)
	}

	fmt.Println()
	fmt.Println("success_rate_curve:")
	keys := make([]seriesKey, 0, len(bySeries))
	for k := range bySeries {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].mode != keys[j].mode {
			return keys[i].mode < keys[j].mode
		}
		if keys[i].poolWorkers != keys[j].poolWorkers {
			return keys[i].poolWorkers < keys[j].poolWorkers
		}
		return keys[i].poolQueue < keys[j].poolQueue
	})
	for _, k := range keys {
		pts := bySeries[k]
		sort.Slice(pts, func(i, j int) bool { return pts[i].concurrency < pts[j].concurrency })
		fmt.Printf("series mode=%s workers=%d queue=%d:", k.mode, k.poolWorkers, k.poolQueue)
		for _, p := range pts {
			fmt.Printf(" (c=%d, ok=%.0f, sr=%.1f%%)", p.concurrency, p.okPerSec, 100*p.successRate)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("throughput_latency_line:")
	for _, k := range keys {
		pts := bySeries[k]
		sort.Slice(pts, func(i, j int) bool { return pts[i].concurrency < pts[j].concurrency })
		fmt.Printf("series mode=%s workers=%d queue=%d:", k.mode, k.poolWorkers, k.poolQueue)
		for _, p := range pts {
			fmt.Printf(" (c=%d, ok=%.0f, p95=%.2fms)", p.concurrency, p.okPerSec, float64(p.p95)/float64(time.Millisecond))
		}
		fmt.Println()
	}
}
