// Package event provides a simple synchronous/async event dispatcher.
package event

import (
	"sync"
)

// Handler is a function that receives an event payload.
type Handler func(payload interface{})

var (
	mu       sync.RWMutex
	handlers = map[string][]Handler{}
)

// Listen registers a handler for the given event name.
func Listen(event string, handler Handler) {
	mu.Lock()
	defer mu.Unlock()
	handlers[event] = append(handlers[event], handler)
}

// Fire dispatches an event synchronously to all registered listeners.
func Fire(event string, payload interface{}) {
	mu.RLock()
	hs := make([]Handler, len(handlers[event]))
	copy(hs, handlers[event])
	mu.RUnlock()

	for _, h := range hs {
		h(payload)
	}
}

// FireAsync dispatches the event to all listeners concurrently.
// It returns immediately without waiting for handlers to complete.
func FireAsync(event string, payload interface{}) {
	mu.RLock()
	hs := make([]Handler, len(handlers[event]))
	copy(hs, handlers[event])
	mu.RUnlock()

	for _, h := range hs {
		go h(payload)
	}
}

// Flush removes all listeners (useful in tests).
func Flush() {
	mu.Lock()
	defer mu.Unlock()
	handlers = map[string][]Handler{}
}
