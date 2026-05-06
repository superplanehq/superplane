package circuitbreaker
import (
"testing"
"time"
)
func TestCircuitBreakerBasic(t *testing.T) {
config := Config{
FailureThreshold:   3,
SuccessThreshold:   2,
Timeout:            100 * time.Millisecond,
FailureStatusCodes: []int{500, 503},
}
cb := NewCircuitBreaker(config)
if cb.State() != StateClosed {
t.Fatalf("initial state should be CLOSED, got %v", cb.State())
}
if !cb.CanAttempt() {
t.Fatal("CLOSED state should allow attempts")
}
cb.RecordFailure(500)
cb.RecordFailure(503)
cb.RecordFailure(500)
if cb.State() != StateOpen {
t.Fatalf("after 3 failures, state should be OPEN, got %v", cb.State())
}
time.Sleep(150 * time.Millisecond)
cb.RecordSuccess()
cb.RecordSuccess()
if cb.State() != StateClosed {
t.Fatalf("after 2 successes in HALF_OPEN, state should be CLOSED, got %v", cb.State())
}
}
func TestCircuitBreakerStatusCodeFiltering(t *testing.T) {
config := Config{
FailureThreshold:   2,
SuccessThreshold:   1,
Timeout:            100 * time.Millisecond,
FailureStatusCodes: []int{500, 503},
}
cb := NewCircuitBreaker(config)
cb.RecordFailure(404)
if cb.State() != StateClosed {
t.Fatal("404 should not count as failure")
}
cb.RecordFailure(500)
cb.RecordFailure(503)
if cb.State() != StateOpen {
t.Fatalf("after 2 real failures, state should be OPEN, got %v", cb.State())
}
}
func TestManagerBasic(t *testing.T) {
config := DefaultConfig()
m := NewManager(config)
breaker1, _ := m.GetBreaker("http://service1.example.com/api")
breaker2, _ := m.GetBreaker("http://service2.example.com/api")
if breaker1.State() != StateClosed {
t.Fatal("new breaker should be CLOSED")
}
breaker1.RecordFailure(500)
if breaker2.State() != StateClosed {
t.Fatal("service2 breaker should be independent")
}
}
func TestManagerStates(t *testing.T) {
config := DefaultConfig()
m := NewManager(config)
_, _ = m.GetBreaker("http://host1.example.com/api")
_, _ = m.GetBreaker("http://host2.example.com/api")
states := m.GetStates()
if len(states) != 2 {
t.Errorf("should have 2 breakers, got %d", len(states))
}
if states["host1.example.com"].State != StateClosed {
t.Error("host1 should be CLOSED")
}
if states["host2.example.com"].State != StateClosed {
t.Error("host2 should be CLOSED")
}
}
func TestManagerReset(t *testing.T) {
config := DefaultConfig()
m := NewManager(config)
breaker, _ := m.GetBreaker("http://service.example.com/api")
for i := 0; i < 5; i++ {
breaker.RecordFailure(500)
}
if breaker.State() != StateOpen {
t.Error("breaker should be OPEN after failures")
}
m.Reset("service.example.com")
breaker, _ = m.GetBreaker("http://service.example.com/api")
if breaker.State() != StateClosed {
t.Error("breaker should be CLOSED after reset")
}
}
func TestCircuitBreakerCanAttemptWithTimeout(t *testing.T) {
config := Config{
FailureThreshold:   1,
SuccessThreshold:   1,
Timeout:            50 * time.Millisecond,
FailureStatusCodes: []int{500},
}
cb := NewCircuitBreaker(config)
cb.RecordFailure(500)
if cb.CanAttempt() {
t.Error("OPEN state should not allow attempts immediately")
}
time.Sleep(25 * time.Millisecond)
if cb.CanAttempt() {
t.Error("before timeout, OPEN should not allow attempts")
}
time.Sleep(30 * time.Millisecond)
if !cb.CanAttempt() {
t.Error("after timeout, OPEN should allow attempts")
}
}
func TestManagerInvalidURL(t *testing.T) {
m := NewManager(DefaultConfig())
_, err := m.GetBreaker("not a valid url")
if err == nil {
t.Error("should return error for invalid URL")
}
}
