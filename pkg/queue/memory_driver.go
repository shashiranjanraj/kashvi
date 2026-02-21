package queue

import (
	"context"
	"sync"
)

// MemoryDriver is an in-process, channel-backed queue driver.
// Perfect for development and testing; not durable across restarts.
type MemoryDriver struct {
	mu sync.Mutex
	ch chan []byte
}

// NewMemoryDriver creates an in-memory queue with a buffer of 1000 jobs.
func NewMemoryDriver() *MemoryDriver {
	return &MemoryDriver{ch: make(chan []byte, 1000)}
}

func (d *MemoryDriver) Push(payload []byte) error {
	d.ch <- payload
	return nil
}

func (d *MemoryDriver) Pop(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case payload := <-d.ch:
		return payload, nil
	}
}
