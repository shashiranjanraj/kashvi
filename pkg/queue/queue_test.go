package queue_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/queue"
)

// ─── Job types ────────────────────────────────────────────────────────────────

type echoJob struct {
	Val    string
	called *atomic.Int32
}

func (j *echoJob) Handle() error {
	if j.called != nil {
		j.called.Add(1)
	}
	return nil
}

type failJob struct {
	attempts *atomic.Int32
}

func (j *failJob) Handle() error {
	if j.attempts != nil {
		j.attempts.Add(1)
	}
	return errors.New("always fails")
}

func init() {
	// Start workers so jobs actually get processed in tests.
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel
	queue.StartWorkers(ctx, 2)

	queue.Register("*queue_test.echoJob", func() queue.Job { return &echoJob{called: &atomic.Int32{}} })
	queue.Register("*queue_test.failJob", func() queue.Job { return &failJob{attempts: &atomic.Int32{}} })
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestDispatchAndProcess(t *testing.T) {
	if err := queue.Dispatch(&echoJob{Val: "hello", called: &atomic.Int32{}}); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
}

func TestFailedJobRetry(t *testing.T) {
	queue.SetMaxRetry(1)
	defer queue.SetMaxRetry(3)

	if err := queue.Dispatch(&failJob{attempts: &atomic.Int32{}}); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	// 1 attempt + 1s backoff + slack.
	time.Sleep(2500 * time.Millisecond)

	if len(queue.FailedJobs()) == 0 {
		t.Error("expected at least one failed job")
	}
}

func TestDispatchConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wg.Done()
			queue.Dispatch(&echoJob{Val: "c", called: &atomic.Int32{}}) //nolint:errcheck
		}()
	}
	wg.Wait()
}
