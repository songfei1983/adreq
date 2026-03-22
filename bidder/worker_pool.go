package bidder

import (
	"context"
	"sync"
)

type WorkerPool struct {
	jobQueue   chan jobRequest
	numWorkers int
	wg         sync.WaitGroup
	mu         sync.RWMutex
	closed     bool
	closeOnce  sync.Once
}

func NewWorkerPool(numWorkers, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		jobQueue:   make(chan jobRequest, queueSize),
		numWorkers: numWorkers,
	}

	for i := 0; i < numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for req := range wp.jobQueue {
		ctx := req.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		req.job(ctx)
	}
}

func (wp *WorkerPool) Submit(ctx context.Context, job Job) bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if wp.closed {
		return false
	}
	select {
	case wp.jobQueue <- jobRequest{ctx: ctx, job: job}:
		return true
	default:
		return false
	}
}

func (wp *WorkerPool) Close() {
	wp.closeOnce.Do(func() {
		wp.mu.Lock()
		defer wp.mu.Unlock()

		wp.closed = true
		close(wp.jobQueue)
	})
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
