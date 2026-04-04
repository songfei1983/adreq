package executor

import (
	"context"
	"errors"
	"sync"
)

type Executor interface {
	Context() context.Context
	Go(task func(ctx context.Context) error) bool
	Wait() error
}

type Group struct {
	ctx    context.Context
	cancel context.CancelFunc

	sem chan struct{}

	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
}

func NewGroup(ctx context.Context, maxConcurrent int) *Group {
	if ctx == nil {
		ctx = context.Background()
	}
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	derivedCtx, cancel := context.WithCancel(ctx)
	return &Group{
		ctx:    derivedCtx,
		cancel: cancel,
		sem:    make(chan struct{}, maxConcurrent),
	}
}

func (g *Group) Context() context.Context {
	return g.ctx
}

func (g *Group) Go(task func(ctx context.Context) error) bool {
	if task == nil {
		return false
	}
	if err := g.ctx.Err(); err != nil {
		return false
	}

	select {
	case g.sem <- struct{}{}:
	case <-g.ctx.Done():
		return false
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer func() { <-g.sem }()

		if err := task(g.ctx); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				g.cancel()
			})
		}
	}()

	return true
}

func (g *Group) Wait() error {
	g.wg.Wait()
	g.cancel()

	if g.err != nil {
		return g.err
	}

	if err := g.ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

type Pool interface {
	Submit(ctx context.Context, job func(context.Context)) bool
	Close()
	Wait()
}

var ErrRejected = errors.New("executor: task rejected")

type PoolExecutor struct {
	ctx    context.Context
	cancel context.CancelFunc

	pool Pool

	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
}

func NewPoolExecutor(ctx context.Context, pool Pool) *PoolExecutor {
	if ctx == nil {
		ctx = context.Background()
	}
	derivedCtx, cancel := context.WithCancel(ctx)
	return &PoolExecutor{
		ctx:    derivedCtx,
		cancel: cancel,
		pool:   pool,
	}
}

func (e *PoolExecutor) Context() context.Context {
	return e.ctx
}

func (e *PoolExecutor) Go(task func(ctx context.Context) error) bool {
	if e.pool == nil || task == nil {
		return false
	}
	if err := e.ctx.Err(); err != nil {
		return false
	}

	e.wg.Add(1)
	ok := e.pool.Submit(e.ctx, func(jobCtx context.Context) {
		defer e.wg.Done()

		if jobCtx == nil {
			jobCtx = e.ctx
		}

		if err := task(jobCtx); err != nil {
			e.errOnce.Do(func() {
				e.err = err
				e.cancel()
			})
		}
	})
	if !ok {
		e.wg.Done()
		e.errOnce.Do(func() {
			e.err = ErrRejected
			e.cancel()
		})
		return false
	}

	return true
}

func (e *PoolExecutor) Wait() error {
	e.wg.Wait()
	e.cancel()

	if e.err != nil {
		return e.err
	}

	if err := e.ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
