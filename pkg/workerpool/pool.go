// Package workerpool provides a bounded goroutine pool with backpressure.
//
// A Pool limits the number of goroutines that can run concurrently, which
// prevents unbounded goroutine creation under bursty load.  When all workers
// are busy, Submit returns ErrPoolFull immediately (non-blocking) so the
// caller can decide to queue, retry, or reject.
//
// Basic usage:
//
//	pool := workerpool.New(50)
//	defer pool.Shutdown()
//
//	err := pool.Submit(func() {
//	    doExpensiveWork()
//	})
//	if errors.Is(err, workerpool.ErrPoolFull) {
//	    // Handle backpressure: return 429, enqueue in Redis, etc.
//	}
package workerpool

import (
	"errors"
	"sync"
)

// ErrPoolFull is returned by Submit when all workers are busy and the task
// queue is at capacity.
var ErrPoolFull = errors.New("workerpool: pool is full")

// ErrPoolClosed is returned by Submit after Shutdown has been called.
var ErrPoolClosed = errors.New("workerpool: pool is closed")

// Pool is a bounded goroutine pool.
type Pool struct {
	tasks   chan func()
	wg      sync.WaitGroup
	once    sync.Once
	closeCh chan struct{}
}

// New creates a Pool with the given number of workers.
// size must be > 0.
func New(size int) *Pool {
	if size <= 0 {
		size = 1
	}

	p := &Pool{
		// Buffer equal to 2× the worker count so bursts can be absorbed.
		tasks:   make(chan func(), size*2),
		closeCh: make(chan struct{}),
	}

	for i := 0; i < size; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	return p
}

// Submit enqueues task for execution.
// It returns immediately — it never blocks.
//   - Returns ErrPoolFull if the task queue is at capacity.
//   - Returns ErrPoolClosed if Shutdown has been called.
func (p *Pool) Submit(task func()) error {
	select {
	case <-p.closeCh:
		return ErrPoolClosed
	default:
	}

	select {
	case p.tasks <- task:
		return nil
	default:
		return ErrPoolFull
	}
}

// SubmitWait is like Submit but blocks until a slot is available or the pool
// is closed.  Returns ErrPoolClosed if the pool is shutting down.
func (p *Pool) SubmitWait(task func()) error {
	select {
	case <-p.closeCh:
		return ErrPoolClosed
	case p.tasks <- task:
		return nil
	}
}

// Shutdown stops accepting new tasks, waits for all in-flight tasks to
// complete, and releases all worker goroutines.
// It is safe to call multiple times.
func (p *Pool) Shutdown() {
	p.once.Do(func() {
		close(p.closeCh)
		close(p.tasks)
		p.wg.Wait()
	})
}

// worker drains the task channel until it is closed.
func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		safeRun(task)
	}
}

// safeRun executes task, recovering from panics so a bad task doesn't kill
// the worker goroutine.
func safeRun(task func()) {
	defer func() { recover() }() //nolint:errcheck
	task()
}
