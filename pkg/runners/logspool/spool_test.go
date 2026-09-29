package logspool

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSpoolUploadsSequentialNDJSONAndWaitsForAcknowledgement(t *testing.T) {
	var mu sync.Mutex
	var sequences []int
	var chunks [][]byte
	releaseFirst := make(chan struct{})
	server := httptest.NewServer(logHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence, err := strconv.Atoi(r.PathValue("sequence"))
		if err != nil {
			t.Errorf("sequence: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		content, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if sequence == 0 {
			<-releaseFirst
		}
		mu.Lock()
		sequences = append(sequences, sequence)
		chunks = append(chunks, content)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})))
	defer server.Close()

	spool, err := New(Config{
		Directory:   t.TempDir(),
		TaskID:      "task-1",
		BaseURL:     server.URL,
		AccessToken: "access-token",
		ChunkBytes:  80,
		MaxBytes:    1024,
		RetryDelay:  10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := spool.Write([]byte("first line\nsecond line\nthird line\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := spool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- spool.Wait(t.Context())
	}()
	select {
	case err := <-waitDone:
		t.Fatalf("Wait returned before acknowledgement: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseFirst)
	if err := <-waitDone; err != nil {
		t.Fatalf("Wait: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(sequences) < 2 {
		t.Fatalf("sequences = %v, want multiple chunks", sequences)
	}
	for i, sequence := range sequences {
		if sequence != i {
			t.Fatalf("sequences = %v", sequences)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(chunks[i])), "\n") {
			if !json.Valid([]byte(line)) {
				t.Fatalf("chunk %d contains invalid NDJSON line %q", i, line)
			}
		}
	}
}

func TestSpoolRetriesBeforeAdvancingSequence(t *testing.T) {
	var mu sync.Mutex
	var attempts []int
	server := httptest.NewServer(logHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence, _ := strconv.Atoi(r.PathValue("sequence"))
		mu.Lock()
		attempts = append(attempts, sequence)
		attempt := len(attempts)
		mu.Unlock()
		if attempt == 1 {
			http.Error(w, "try again", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})))
	defer server.Close()

	spool, err := New(Config{
		Directory:   t.TempDir(),
		TaskID:      "task-2",
		BaseURL:     server.URL,
		AccessToken: "access-token",
		ChunkBytes:  32,
		MaxBytes:    1024,
		RetryDelay:  10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, _ = spool.Write([]byte("hello\n"))
	if err := spool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := spool.Wait(t.Context()); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(attempts) != 2 || attempts[0] != 0 || attempts[1] != 0 {
		t.Fatalf("attempts = %v", attempts)
	}
}

func TestSpoolUploadsPartialChunkWhileTaskIsRunning(t *testing.T) {
	uploaded := make(chan struct{}, 1)
	server := httptest.NewServer(logHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		uploaded <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	})))
	defer server.Close()

	spool, err := New(Config{
		Directory:     t.TempDir(),
		TaskID:        "task-running",
		BaseURL:       server.URL,
		AccessToken:   "access-token",
		ChunkBytes:    1024,
		MaxBytes:      4096,
		RetryDelay:    10 * time.Millisecond,
		FlushInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, _ = spool.Write([]byte("partial chunk\n"))

	select {
	case <-uploaded:
	case <-time.After(time.Second):
		t.Fatal("partial chunk was not uploaded while the task was running")
	}
	if err := spool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := spool.Wait(t.Context()); err != nil {
		t.Fatalf("Wait: %v", err)
	}
}

func TestSpoolDefaultFlushIntervalUsesJitterRange(t *testing.T) {
	spool := &Spool{}
	for range 100 {
		interval := spool.nextFlushInterval()
		if interval < defaultFlushIntervalMin || interval > defaultFlushIntervalMax {
			t.Fatalf(
				"flush interval = %s, want between %s and %s",
				interval,
				defaultFlushIntervalMin,
				defaultFlushIntervalMax,
			)
		}
	}
}

func TestSpoolBoundsDiskUseAndWritesDroppedMarker(t *testing.T) {
	var mu sync.Mutex
	var uploaded strings.Builder
	server := httptest.NewServer(logHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := io.ReadAll(r.Body)
		mu.Lock()
		uploaded.Write(content)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})))
	defer server.Close()

	spool, err := New(Config{
		Directory:   t.TempDir(),
		TaskID:      "task-3",
		BaseURL:     server.URL,
		AccessToken: "access-token",
		ChunkBytes:  128,
		MaxBytes:    256,
		RetryDelay:  10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, _ = spool.Write([]byte(strings.Repeat("large output ", 200) + "\n"))
	if err := spool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := spool.Wait(t.Context()); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if spool.PeakBytes() > 256 {
		t.Fatalf("peak bytes = %d", spool.PeakBytes())
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(uploaded.String(), `"text":"[Runner logs were dropped`) {
		t.Fatalf("missing dropped marker in %q", uploaded.String())
	}
}

func logHandler(next http.Handler) http.Handler {
	router := http.NewServeMux()
	router.Handle(
		"PUT /runner/v1/tasks/{task_id}/logs/chunks/{sequence}",
		next,
	)
	return router
}

func TestSpoolWaitHonorsContext(t *testing.T) {
	server := httptest.NewServer(logHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	})))
	defer server.Close()

	spool, err := New(Config{
		Directory:   t.TempDir(),
		TaskID:      "task-4",
		BaseURL:     server.URL,
		AccessToken: "access-token",
		ChunkBytes:  64,
		MaxBytes:    1024,
		RetryDelay:  10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, _ = spool.Write([]byte("hello\n"))
	if err := spool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := spool.Wait(ctx); err == nil {
		t.Fatal("expected context error")
	}
}
