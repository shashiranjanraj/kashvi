package workerpool_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/workerpool"
)

func TestPool_SubmitAndExecute(t *testing.T) {
	pool := workerpool.New(4)
	defer pool.Shutdown()

	const n = 100
	var count atomic.Int64

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		err := pool.SubmitWait(func() {
			defer wg.Done()
			count.Add(1)
		})
		if err != nil {
			t.Fatalf("SubmitWait returned unexpected error: %v", err)
		}
	}

	wg.Wait()

	if got := count.Load(); got != n {
		t.Errorf("expected %d tasks to run, got %d", n, got)
	}
}

func TestPool_ErrPoolFull(t *testing.T) {
	// Size-1 pool whose only worker is blocked.
	pool := workerpool.New(1)
	defer pool.Shutdown()

	blocker := make(chan struct{})
	submitted := make(chan struct{})

	// Block the single worker.
	_ = pool.SubmitWait(func() {
		close(submitted)
		<-blocker
	})
	<-submitted

	// Fill the 2-slot queue (buffer = 2× worker count = 2).
	_ = pool.Submit(func() {})
	_ = pool.Submit(func() {})

	// Now the queue is full — Submit must return ErrPoolFull.
	err := pool.Submit(func() {})
	if !errors.Is(err, workerpool.ErrPoolFull) {
		t.Errorf("expected ErrPoolFull, got %v", err)
	}

	close(blocker) // unblock the worker
}

func TestPool_ErrPoolClosed(t *testing.T) {
	pool := workerpool.New(2)
	pool.Shutdown()

	err := pool.Submit(func() {})
	if !errors.Is(err, workerpool.ErrPoolClosed) {
		t.Errorf("expected ErrPoolClosed after Shutdown, got %v", err)
	}
}

func TestPool_PanicRecovery(t *testing.T) {
	pool := workerpool.New(2)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	wg.Add(1)

	// A panicking task must not kill the worker or block subsequent tasks.
	_ = pool.SubmitWait(func() {
		defer wg.Done()
		panic("intentional panic — should be recovered")
	})

	wg.Wait()

	// Pool must still accept new tasks after recovering from a panic.
	normal := make(chan struct{})
	_ = pool.SubmitWait(func() { close(normal) })

	select {
	case <-normal:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("pool did not recover from panic — subsequent task never ran")
	}
}

func TestPool_Shutdown_NoGoroutineLeak(t *testing.T) {
	pool := workerpool.New(10)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		_ = pool.SubmitWait(func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
		})
	}

	wg.Wait()
	pool.Shutdown() // must return promptly without leaking goroutines
}
