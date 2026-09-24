package broker

import "sync"

// WaitHub wakes all registered waiters when Notify is called (e.g. after a task is enqueued).
// Each waiter uses a dedicated buffered channel (capacity 1) so Notify coalesces per waiter.
// Safe for concurrent Notify and Register/Unregister.
type WaitHub struct {
	mu        sync.Mutex
	listeners []chan struct{}
}

// NewWaitHub returns a hub with no listeners until Register is called.
func NewWaitHub() *WaitHub {
	return &WaitHub{}
}

// Register adds a waiter channel. Call Unregister with the returned channel when done.
func (h *WaitHub) Register() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.listeners = append(h.listeners, ch)
	h.mu.Unlock()
	return ch
}

// Unregister removes a waiter channel (no-op if unknown).
func (h *WaitHub) Unregister(ch chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, c := range h.listeners {
		if c == ch {
			h.listeners = append(h.listeners[:i], h.listeners[i+1:]...)
			return
		}
	}
}

// Notify wakes every registered listener (non-blocking per listener; drops if buffer full).
func (h *WaitHub) Notify() {
	h.mu.Lock()
	listeners := append([]chan struct{}(nil), h.listeners...)
	h.mu.Unlock()
	for _, ch := range listeners {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
