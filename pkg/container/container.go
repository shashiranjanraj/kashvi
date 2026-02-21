// Package container provides a lightweight dependency injection container.
package container

import (
	"fmt"
	"sync"
)

// Factory is a function that produces a service instance.
type Factory func() interface{}

var (
	mu         sync.RWMutex
	bindings   = map[string]Factory{}
	singletons = map[string]interface{}{}
)

// Bind registers a factory under key. Each call to Make invokes factory anew.
func Bind(key string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	bindings[key] = factory
}

// Singleton registers a factory that is called once; subsequent Make calls
// return the cached instance.
func Singleton(key string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	bindings[key] = factory
	// Mark as singleton by reserving the slot (resolved lazily on first Make).
	singletons[key] = nil
}

// Make resolves and returns the service registered under key.
// Panics if the key has not been bound (same behaviour as Laravel's container).
func Make(key string) interface{} {
	mu.Lock()
	defer mu.Unlock()

	// Singleton already instantiated?
	if inst, ok := singletons[key]; ok && inst != nil {
		return inst
	}

	factory, ok := bindings[key]
	if !ok {
		panic(fmt.Sprintf("kashvi/container: unknown binding %q", key))
	}

	instance := factory()

	// Cache if this is a singleton slot.
	if _, isSingleton := singletons[key]; isSingleton {
		singletons[key] = instance
	}

	return instance
}

// Has reports whether a key has been bound.
func Has(key string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := bindings[key]
	return ok
}
