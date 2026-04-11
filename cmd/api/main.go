package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/songfei1983/adreq/bidder"
	"github.com/songfei1983/adreq/filter"
	"github.com/songfei1983/adreq/infra"
	"github.com/songfei1983/adreq/model"
	"github.com/songfei1983/adreq/server"
)

func envInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(name string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func envString(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}

func main() {
	var (
		addr            = flag.String("addr", envString("ADREQ_ADDR", ":8080"), "listen address")
		mode            = flag.String("mode", envString("ADREQ_MODE", "direct"), "direct|pool")
		timeout         = flag.Duration("timeout", envDuration("ADREQ_TIMEOUT", 100*time.Millisecond), "request timeout")
		maxRequests     = flag.Int("max-requests", envInt("ADREQ_MAX_REQUESTS", 0), "max concurrent in-flight requests (0 disables)")
		admission       = flag.String("admission", envString("ADREQ_ADMISSION", "reject"), "reject|block")
		maxImpConc      = flag.Int("max-imp-conc", envInt("ADREQ_MAX_IMP_CONC", 0), "max concurrent imps per request (0 uses len(imps))")
		maxCandConc     = flag.Int("max-cand-conc", envInt("ADREQ_MAX_CAND_CONC", 0), "max concurrent candidates per request (0 uses max-imp-conc)")
		poolWorkers     = flag.Int("pool-workers", envInt("ADREQ_POOL_WORKERS", 16), "worker pool size (mode=pool)")
		poolQueue       = flag.Int("pool-queue", envInt("ADREQ_POOL_QUEUE", 65536), "worker pool queue size (mode=pool)")
		readHeaderTo    = flag.Duration("read-header-timeout", envDuration("ADREQ_READ_HEADER_TIMEOUT", 2*time.Second), "http server read header timeout")
		shutdownTimeout = flag.Duration("shutdown-timeout", envDuration("ADREQ_SHUTDOWN_TIMEOUT", 5*time.Second), "graceful shutdown timeout")
	)
	flag.Parse()

	var admissionMode server.RequestAdmissionMode
	switch strings.ToLower(strings.TrimSpace(*admission)) {
	case "", "reject":
		admissionMode = server.RequestAdmissionReject
	case "block":
		admissionMode = server.RequestAdmissionBlock
	default:
		fmt.Fprintln(os.Stderr, "invalid -admission (want reject|block)")
		os.Exit(2)
	}

	budget := infra.NewBudgetStore()
	budget.Set("camp_0", 1000000)
	budget.Set("camp_1", 1000000)
	budget.Set("camp_2", 1000000)

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

	s := server.NewAdServer(source, processor, server.Config{
		RequestTimeout:          *timeout,
		MaxConcurrentRequests:   *maxRequests,
		RequestAdmissionMode:    admissionMode,
		MaxConcurrentImps:       *maxImpConc,
		MaxConcurrentCandidates: *maxCandConc,
		Budget:                  budget,
	})
	if strings.ToLower(strings.TrimSpace(*mode)) == "pool" {
		s = s.WithWorkerPool(infra.NewWorkerPool(*poolWorkers, *poolQueue))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/bid", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if len(raw) == 0 {
			http.Error(w, "empty body", http.StatusBadRequest)
			return
		}

		var wrapper map[string]json.RawMessage
		if err := json.Unmarshal(raw, &wrapper); err == nil {
			if inner, ok := wrapper["request"]; ok && len(inner) > 0 {
				raw = inner
			}
		}

		var br model.BidRequest
		if err := json.Unmarshal(raw, &br); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		var (
			resp any
			err  error
		)
		if s == nil {
			http.Error(w, "server not ready", http.StatusServiceUnavailable)
			return
		}

		if strings.ToLower(strings.TrimSpace(*mode)) == "pool" {
			resp, err = s.HandleRequestWithPool(ctx, &br)
		} else {
			resp, err = s.HandleRequest(ctx, &br)
		}
		if err != nil {
			if errors.Is(err, server.ErrOverloaded) {
				http.Error(w, "overloaded", http.StatusTooManyRequests)
				return
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				http.Error(w, "timeout", http.StatusGatewayTimeout)
				return
			}
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		_ = enc.Encode(resp)
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: *readHeaderTo,
	}

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(ctx)
		s.Shutdown()
	}()

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
