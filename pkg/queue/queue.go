// Package queue provides a background job processing system for Kashvi.
//
// Usage:
//
//	// Define a job
//	type WelcomeEmailJob struct { UserID uint }
//	func (j WelcomeEmailJob) Handle() error {
//	    log.Println("Sending welcome email to user", j.UserID)
//	    return nil
//	}
//
//	// Dispatch
//	queue.Dispatch(WelcomeEmailJob{UserID: 1})
//	queue.DispatchAfter(WelcomeEmailJob{UserID: 2}, 30*time.Second)
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/logger"
)

// Job is the interface every queued job must satisfy.
type Job interface {
	// Handle executes the job. Return a non-nil error to signal failure.
	Handle() error
}

// FailedJob holds information about a job that failed.
type FailedJob struct {
	Job      Job
	Err      error
	FailedAt time.Time
	Attempts int
}

// Driver is the queue storage backend.
type Driver interface {
	Push(payload []byte) error
	Pop(ctx context.Context) ([]byte, error)
}

// ------------------- Manager -------------------

// Manager is the central queue hub.
type Manager struct {
	mu       sync.RWMutex
	driver   Driver
	registry map[string]func() Job // type name → constructor
	failed   []FailedJob
	maxRetry int
}

var defaultManager = &Manager{
	registry: map[string]func() Job{},
	maxRetry: 3,
	driver:   NewMemoryDriver(),
}

// SetDriver swaps the underlying queue driver (e.g. Redis).
func SetDriver(d Driver) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	defaultManager.driver = d
}

// SetMaxRetry sets how many times a failing job is retried.
func SetMaxRetry(n int) { defaultManager.maxRetry = n }

// Register makes a job type available for deserialization by name.
// Call this once at boot for every job type you define.
func Register(name string, factory func() Job) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	defaultManager.registry[name] = factory
}

// ------------------- Dispatch -------------------

type envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Dispatch pushes job onto the default queue immediately.
func Dispatch(job Job) error {
	return defaultManager.push(job)
}

// DispatchAfter pushes job onto the queue after a delay.
// Note: for the in-memory driver, this spawns a goroutine; for Redis, use a
// dedicated delayed-queue strategy (e.g. sorted set).
func DispatchAfter(job Job, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		if err := Dispatch(job); err != nil {
			logger.Error("queue: delayed dispatch failed", "error", err)
		}
	}()
}

func (m *Manager) push(job Job) error {
	typeName := fmt.Sprintf("%T", job)

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("queue: marshal job %s: %w", typeName, err)
	}

	env, err := json.Marshal(envelope{Type: typeName, Payload: payload})
	if err != nil {
		return fmt.Errorf("queue: marshal envelope: %w", err)
	}

	m.mu.RLock()
	d := m.driver
	m.mu.RUnlock()

	return d.Push(env)
}

// ------------------- Worker -------------------

// StartWorkers launches n concurrent workers that process jobs from the queue.
// The workers run until ctx is cancelled.
func StartWorkers(ctx context.Context, n int) {
	for i := 0; i < n; i++ {
		go defaultManager.work(ctx)
	}
	logger.Info("queue: workers started", "count", n)
}

func (m *Manager) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			m.mu.RLock()
			d := m.driver
			m.mu.RUnlock()

			raw, err := d.Pop(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return // context cancelled
				}
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if raw == nil {
				continue
			}

			m.process(raw)
		}
	}
}

func (m *Manager) process(raw []byte) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		logger.Error("queue: bad envelope", "error", err)
		return
	}

	m.mu.RLock()
	factory, ok := m.registry[env.Type]
	m.mu.RUnlock()

	if !ok {
		// Job type not registered — run via rawJob fallback
		logger.Warn("queue: unregistered job type", "type", env.Type)
		return
	}

	job := factory()
	if err := json.Unmarshal(env.Payload, job); err != nil {
		logger.Error("queue: unmarshal payload", "type", env.Type, "error", err)
		return
	}

	m.runWithRetry(job, env.Type)
}

func (m *Manager) runWithRetry(job Job, typeName string) {
	var lastErr error
	for attempt := 1; attempt <= m.maxRetry; attempt++ {
		if err := job.Handle(); err != nil {
			lastErr = err
			logger.Warn("queue: job failed, retrying",
				"type", typeName, "attempt", attempt, "error", err)
			time.Sleep(time.Duration(attempt) * time.Second) // backoff
			continue
		}
		logger.Info("queue: job processed", "type", typeName)
		return
	}

	// All retries exhausted — persist the failure.
	m.persistFailed(job, typeName, lastErr, m.maxRetry)
	logger.Error("queue: job exhausted retries", "type", typeName, "error", lastErr)
}

// FailedJobs returns a snapshot of all failed jobs.
func FailedJobs() []FailedJob {
	defaultManager.mu.RLock()
	defer defaultManager.mu.RUnlock()
	out := make([]FailedJob, len(defaultManager.failed))
	copy(out, defaultManager.failed)
	return out
}
