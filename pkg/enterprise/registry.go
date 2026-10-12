package enterprise

import (
	"fmt"
	"sync"
)

// Key names an Enterprise capability. It is not a license feature. One
// capability can check several license features.
type Key string

const (
	// RBAC is the custom role and group implementation. The license features
	// custom_roles and groups are checked inside that implementation.
	RBAC Key = "rbac"
)

// Registry maps a capability name to the code that implements it. Callers
// resolve an implementation with Get. This is the same lookup shape as
// pkg/registry, which returns an action or a trigger by name.
type Registry struct {
	mu     sync.RWMutex
	values map[Key]any
}

// NewRegistry registers the Community implementation of each capability. A
// Community implementation refuses the operation.
func NewRegistry() *Registry {
	registry := &Registry{values: map[Key]any{}}
	registry.Register(RBAC, communityRbac{})
	return registry
}

// Register stores the implementation for a capability. A later registration
// replaces the previous one.
func (r *Registry) Register(key Key, implementation any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = implementation
}

// Get returns the implementation registered for key. Go has no generic
// methods, so this is a function. A missing key or a value of the wrong type
// is an error.
func Get[T any](registry *Registry, key Key) (T, error) {
	var zero T
	if registry == nil {
		return zero, fmt.Errorf("enterprise capability %s is not registered", key)
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	value, ok := registry.values[key]
	if !ok {
		return zero, fmt.Errorf("enterprise capability %s is not registered", key)
	}

	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("enterprise capability %s has the wrong type", key)
	}

	return typed, nil
}
