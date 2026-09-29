package brokerlive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	flushInterval = 200 * time.Millisecond
	flushBytes    = 32 * 1024
	maxPostBytes  = 1 << 20
	closeTimeout  = 5 * time.Second
)

// Writer posts task stdout/stderr chunks to task-broker live-log ingest.
type Writer struct {
	client   *http.Client
	endpoint string
	token    string

	mu        sync.Mutex
	buf       []byte
	closed    bool
	stopFlush chan struct{}
	flushDone chan struct{}
}

// NewWriter starts a background flush loop. Close flushes remaining bytes.
func NewWriter(client *http.Client, baseURL, token, taskID string) *Writer {
	if client == nil {
		client = http.DefaultClient
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	w := &Writer{
		client:    client,
		endpoint:  base + "/v1/tasks/" + strings.TrimSpace(taskID) + "/live-log-events",
		token:     strings.TrimSpace(token),
		stopFlush: make(chan struct{}),
		flushDone: make(chan struct{}),
	}
	go w.flushLoop()
	return w
}

func (w *Writer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	w.buf = append(w.buf, p...)
	var chunk []byte
	if len(w.buf) >= flushBytes {
		chunk = w.takeLocked()
	}
	w.mu.Unlock()
	if len(chunk) > 0 {
		if err := w.post(context.Background(), chunk); err != nil {
			return len(p), err
		}
	}
	return len(p), nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	close(w.stopFlush)
	<-w.flushDone
	return nil
}

func (w *Writer) flushLoop() {
	t := time.NewTicker(flushInterval)
	defer t.Stop()
	defer close(w.flushDone)
	for {
		select {
		case <-w.stopFlush:
			ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
			_ = w.flush(ctx)
			cancel()
			return
		case <-t.C:
			_ = w.flush(context.Background())
		}
	}
}

func (w *Writer) flush(ctx context.Context) error {
	w.mu.Lock()
	chunk := w.takeLocked()
	w.mu.Unlock()
	if len(chunk) == 0 {
		return nil
	}
	return w.post(ctx, chunk)
}

func (w *Writer) takeLocked() []byte {
	if len(w.buf) == 0 {
		return nil
	}
	chunk := w.buf
	if len(chunk) > maxPostBytes {
		chunk = w.buf[:maxPostBytes]
		w.buf = w.buf[maxPostBytes:]
		return chunk
	}
	w.buf = nil
	return chunk
}

func (w *Writer) post(ctx context.Context, chunk []byte) error {
	if len(chunk) == 0 {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.endpoint, bytes.NewReader(chunk))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if w.token != "" {
		req.Header.Set("Authorization", "Bearer "+w.token)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("brokerlive: status %d", resp.StatusCode)
	}
	return nil
}
