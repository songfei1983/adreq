package bidder

import (
	"context"
	"sync"
	"sync/atomic"
)

type Job func(ctx context.Context)

type WorkerPool struct {
	jobQueue   chan Job
	numWorkers int
	wg         sync.WaitGroup
	closed     uint32
	closeOnce  sync.Once
}

func NewWorkerPool(numWorkers, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		jobQueue:   make(chan Job, queueSize),
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
	for job := range wp.jobQueue {
		job(context.Background())
	}
}

func (wp *WorkerPool) Submit(job Job) bool {
	if atomic.LoadUint32(&wp.closed) == 1 {
		return false
	}
	select {
	case wp.jobQueue <- job:
		return true
	default:
		return false
	}
}

func (wp *WorkerPool) SubmitBlocking(job Job) {
	wp.jobQueue <- job
}

func (wp *WorkerPool) Close() {
	wp.closeOnce.Do(func() {
		atomic.StoreUint32(&wp.closed, 1)
		close(wp.jobQueue)
	})
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

type JobWithResult struct {
	Job    func(ctx context.Context) interface{}
	Result chan interface{}
}

type ResultCollector struct {
	jobs    []JobWithResult
	result  chan interface{}
	wg      sync.WaitGroup
	results []interface{}
	mu      sync.Mutex
}

func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		result: make(chan interface{}),
	}
}

func (rc *ResultCollector) AddJob(job func(ctx context.Context) interface{}) {
	rc.jobs = append(rc.jobs, JobWithResult{
		Job:    job,
		Result: make(chan interface{}, 1),
	})
	rc.wg.Add(1)
}

func (rc *ResultCollector) Process(ctx context.Context) []interface{} {
	for i := range rc.jobs {
		go func(j JobWithResult) {
			defer rc.wg.Done()
			select {
			case <-ctx.Done():
				j.Result <- nil
			case result := <-rc.result:
				j.Result <- result
			}
		}(rc.jobs[i])
	}

	rc.wg.Wait()
	close(rc.result)

	var results []interface{}
	for result := range rc.result {
		if result != nil {
			results = append(results, result)
		}
	}
	return results
}
