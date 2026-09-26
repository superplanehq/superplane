package livelogs

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
)

const maxRecordsPerTask = 20000

// Hub stores in-memory live-log records when CloudWatch is not configured.
type Hub struct {
	mu    sync.Mutex
	tasks map[string]*taskBuffer
}

type taskBuffer struct {
	mu        sync.Mutex
	records   []json.RawMessage
	remainder []byte
	closed    bool
	waiters   []chan struct{}
}

// NewHub returns an empty live-log hub.
func NewHub() *Hub {
	return &Hub{tasks: map[string]*taskBuffer{}}
}

func (h *Hub) buffer(taskID string) *taskBuffer {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tasks == nil {
		h.tasks = map[string]*taskBuffer{}
	}
	b, ok := h.tasks[taskID]
	if !ok {
		b = &taskBuffer{}
		h.tasks[taskID] = b
	}
	return b
}

// Append converts a raw stdout/stderr chunk into NDJSON records and notifies waiters.
func (h *Hub) Append(taskID string, chunk []byte) {
	if taskID == "" || len(chunk) == 0 {
		return
	}
	b := h.buffer(taskID)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.remainder = append(b.remainder, chunk...)
	b.drainLines(false)
	b.notifyLocked()
}

// Close flushes a trailing partial line and unblocks Wait callers.
func (h *Hub) Close(taskID string) {
	if taskID == "" {
		return
	}
	b := h.buffer(taskID)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.drainLines(true)
	b.closed = true
	b.notifyLocked()
}

// Wait returns records starting at from. It blocks until records exist, the
// buffer is closed, or ctx is done.
func (h *Hub) Wait(ctx context.Context, taskID string, from int) (records []json.RawMessage, next int, closed bool, err error) {
	if taskID == "" {
		return nil, from, true, nil
	}
	b := h.buffer(taskID)
	notify := make(chan struct{}, 1)

	b.mu.Lock()
	b.waiters = append(b.waiters, notify)
	b.mu.Unlock()
	defer b.unregister(notify)

	for {
		b.mu.Lock()
		if from < 0 {
			from = 0
		}
		if from < len(b.records) {
			out := append([]json.RawMessage(nil), b.records[from:]...)
			next := len(b.records)
			closed := b.closed
			b.mu.Unlock()
			return out, next, closed, nil
		}
		if b.closed {
			b.mu.Unlock()
			return nil, from, true, nil
		}
		b.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, from, false, ctx.Err()
		case <-notify:
		}
	}
}

func (b *taskBuffer) unregister(ch chan struct{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, waiter := range b.waiters {
		if waiter == ch {
			b.waiters = append(b.waiters[:i], b.waiters[i+1:]...)
			return
		}
	}
}

func (b *taskBuffer) notifyLocked() {
	for _, ch := range b.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (b *taskBuffer) drainLines(flushRemainder bool) {
	data := b.remainder
	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}
		line := data[:idx]
		data = data[idx+1:]
		b.appendLine(line)
	}
	if flushRemainder {
		b.appendLine(data)
		b.remainder = nil
		return
	}
	b.remainder = data
}

func (b *taskBuffer) appendLine(line []byte) {
	if len(b.records) >= maxRecordsPerTask {
		return
	}
	text := string(bytes.TrimSpace(line))
	if text == "" {
		return
	}
	raw, err := json.Marshal(RecordFromLogMessage(text))
	if err != nil {
		return
	}
	b.records = append(b.records, raw)
}
