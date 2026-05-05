package circuitbreaker

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

type State string

const (
	StateClosed   State = "CLOSED"
	StateOpen     State = "OPEN"
	StateHalfOpen State = "HALF_OPEN"
)

type Config struct {
	FailureThreshold   int
	SuccessThreshold   int
	Timeout            time.Duration
	FailureStatusCodes []int
}

func DefaultConfig() Config {
	return Config{
		FailureThreshold:   5,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		FailureStatusCodes: []int{500, 503, 502, 504},
	}
}

type CircuitBreaker struct {
	mu sync.RWMutex

	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	config          Config
}

func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
	}
}

func (cb *CircuitBreaker) CanAttempt() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateHalfOpen:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			return true
		}
		return false
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
		cb.successCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	case StateOpen:
		cb.successCount = 1
	}
}

func (cb *CircuitBreaker) RecordFailure(statusCode int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.isFailureStatus(statusCode) {
		if cb.state == StateClosed {
			cb.failureCount = 0
		}
		return
	}

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.failureCount = 1
		cb.successCount = 0
	case StateOpen:
		cb.failureCount++
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) isFailureStatus(statusCode int) bool {
	for _, code := range cb.config.FailureStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

type ManagerState struct {
	State           State
	FailureCount    int
	SuccessCount    int
	LastFailureTime time.Time
}

type Manager struct {
	mu          sync.RWMutex
	breakers    map[string]*CircuitBreaker
	config      Config
	hostConfigs map[string]Config
}

func NewManager(defaultConfig Config) *Manager {
	return &Manager{
		breakers:    make(map[string]*CircuitBreaker),
		hostConfigs: make(map[string]Config),
		config:      defaultConfig,
	}
}

func (m *Manager) GetBreaker(urlStr string) (*CircuitBreaker, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Host
	if host == "" {
		return nil, fmt.Errorf("URL must have a host")
	}

	return m.getBreakerForHost(host), nil
}

func (m *Manager) getBreakerForHost(host string) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if breaker, exists := m.breakers[host]; exists {
		return breaker
	}

	config := m.config
	if hostConfig, exists := m.hostConfigs[host]; exists {
		config = hostConfig
	}

	breaker := NewCircuitBreaker(config)
	m.breakers[host] = breaker
	return breaker
}

func (m *Manager) GetStates() map[string]ManagerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]ManagerState, len(m.breakers))
	for host, breaker := range m.breakers {
		breaker.mu.RLock()
		states[host] = ManagerState{
			State:           breaker.state,
			FailureCount:    breaker.failureCount,
			SuccessCount:    breaker.successCount,
			LastFailureTime: breaker.lastFailureTime,
		}
		breaker.mu.RUnlock()
	}

	return states
}

func (m *Manager) Reset(host string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	config := m.config
	if hostConfig, exists := m.hostConfigs[host]; exists {
		config = hostConfig
	}

	m.breakers[host] = NewCircuitBreaker(config)
}

func (m *Manager) SetHostConfig(host string, config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hostConfigs[host] = config
	if breaker, exists := m.breakers[host]; exists {
		breaker.mu.Lock()
		breaker.config = config
		breaker.failureCount = 0
		breaker.successCount = 0
		breaker.state = StateClosed
		breaker.mu.Unlock()
	}
}
