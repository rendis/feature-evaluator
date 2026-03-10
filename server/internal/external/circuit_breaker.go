package external

import (
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
)

// CBStateChangeFunc is called when a circuit breaker transitions state.
// opened is true when the circuit opens, false when it closes.
type CBStateChangeFunc func(opened bool)

// CircuitBreakerManager manages per-endpoint circuit breakers.
type CircuitBreakerManager struct {
	mu            sync.RWMutex
	breakers      map[string]*gobreaker.CircuitBreaker[[]byte]
	onStateChange CBStateChangeFunc
}

// NewCircuitBreakerManager creates a new circuit breaker manager.
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*gobreaker.CircuitBreaker[[]byte]),
	}
}

// SetOnStateChange sets the callback for circuit breaker state transitions.
func (m *CircuitBreakerManager) SetOnStateChange(fn CBStateChangeFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onStateChange = fn
}

// Get returns the circuit breaker for a given endpoint, creating one if needed.
func (m *CircuitBreakerManager) Get(endpoint string) *gobreaker.CircuitBreaker[[]byte] {
	m.mu.RLock()
	cb, ok := m.breakers[endpoint]
	m.mu.RUnlock()
	if ok {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := m.breakers[endpoint]; ok {
		return cb
	}

	stateChangeFn := m.onStateChange
	cb = gobreaker.NewCircuitBreaker[[]byte](gobreaker.Settings{
		Name:        endpoint,
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			if stateChangeFn == nil {
				return
			}
			if to == gobreaker.StateOpen {
				stateChangeFn(true)
			} else if from == gobreaker.StateOpen {
				stateChangeFn(false)
			}
		},
	})

	m.breakers[endpoint] = cb
	return cb
}

// IsOpen returns true if the circuit breaker for the endpoint is open.
func (m *CircuitBreakerManager) IsOpen(endpoint string) bool {
	m.mu.RLock()
	cb, ok := m.breakers[endpoint]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	return cb.State() == gobreaker.StateOpen
}
