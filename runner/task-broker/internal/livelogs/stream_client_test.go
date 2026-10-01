package livelogs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStreamCloudWatchLogPagesEvents(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	var mu sync.Mutex
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		_, _ = io.Copy(io.Discard, r.Body)
		if r.Header.Get("X-Amz-Target") != "Logs_20140328.GetLogEvents" {
			http.Error(w, "unexpected target "+r.Header.Get("X-Amz-Target"), http.StatusBadRequest)
			return
		}
		mu.Lock()
		calls++
		call := calls
		mu.Unlock()
		if call == 1 {
			_, _ = w.Write([]byte(`{"events":[{"message":"paged line","timestamp":1}],"nextForwardToken":"page-2","nextBackwardToken":"start"}`))
			return
		}
		_, _ = w.Write([]byte(`{"events":[],"nextForwardToken":"page-2","nextBackwardToken":"page-2"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var buf bytes.Buffer
	err := StreamCloudWatchLogToNDJSON(ctx, cancelAfter{&buf, cancel}, nil, "tasks", "task-1", "us-east-1", nil)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"text":"paged line"`) {
		t.Fatalf("output = %s", buf.String())
	}
	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls < 1 {
		t.Fatal("expected GetLogEvents")
	}
}

type cancelAfter struct {
	buf    *bytes.Buffer
	cancel context.CancelFunc
}

func (c cancelAfter) Write(p []byte) (int, error) {
	n, err := c.buf.Write(p)
	c.cancel()
	return n, err
}

func TestCloudWatchHeartbeatDuringStalledRequest(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	requestStarted := make(chan time.Time, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requestStarted <- time.Now():
		default:
		}
		timer := time.NewTimer(16 * time.Second)
		defer timer.Stop()
		select {
		case <-r.Context().Done():
		case <-timer.C:
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			_, _ = w.Write([]byte(`{"events":[],"nextForwardToken":"tail"}`))
		}
	}))
	defer srv.Close()
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	writer := &heartbeatRecorder{cancel: cancel}
	err := StreamCloudWatchLogToNDJSON(ctx, writer, writer, "tasks", "task-1", "us-east-1", nil)
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stream error: %v", err)
	}
	var started time.Time
	select {
	case started = <-requestStarted:
	default:
		t.Fatal("stalled request did not start")
	}
	if writer.pings != 1 {
		t.Fatalf("pings=%d", writer.pings)
	}
	held := writer.pingAt.Sub(started)
	if held >= HeartbeatInterval || held < HeartbeatInterval-requestBudgetMargin-time.Second {
		t.Fatalf("stalled request delayed the heartbeat by %v", held)
	}
}

func TestCloudWatchLogErrorEndsStream(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"__type":"InvalidParameterException","message":"bad log group"}`))
	}))
	defer srv.Close()
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var buf bytes.Buffer
	err := StreamCloudWatchLogToNDJSON(ctx, &buf, nil, "tasks", "task-1", "us-east-1", nil)
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stream error: %v", err)
	}
	if !strings.Contains(buf.String(), `"type":"error"`) {
		t.Fatalf("output = %s", buf.String())
	}
}

func TestCloudWatchHeartbeatDuringQuietTail(t *testing.T) {
	for _, flowing := range []bool{false, true} {
		name := "quiet"
		if flowing {
			name = "flowing"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("AWS_ACCESS_KEY_ID", "test")
			t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
			t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/x-amz-json-1.1")
				if flowing {
					_, _ = w.Write([]byte(`{"events":[{"message":"still running"}],"nextForwardToken":"tail"}`))
					return
				}
				_, _ = w.Write([]byte(`{"events":[],"nextForwardToken":"tail"}`))
			}))
			defer srv.Close()
			t.Setenv("AWS_ENDPOINT_URL", srv.URL)

			ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
			defer cancel()
			writer := &heartbeatRecorder{cancel: cancel}
			started := time.Now()
			isTaskTerminal := func(context.Context) (bool, error) {
				// A failed status lookup must not end a quiet stream.
				return false, errors.New("task store unavailable")
			}
			err := StreamCloudWatchLogToNDJSON(ctx, writer, writer, "tasks", "task-1", "us-east-1", isTaskTerminal)
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("stream error: %v", err)
			}
			if flowing {
				if writer.pings != 0 || writer.flushes == 0 {
					t.Fatalf("flowing stream: pings=%d flushes=%d", writer.pings, writer.flushes)
				}
				return
			}
			if writer.pings != 1 || writer.flushes != 1 {
				t.Fatalf("quiet stream: pings=%d flushes=%d", writer.pings, writer.flushes)
			}
			// Allow a small scheduling margin around the 15-second heartbeat deadline.
			if elapsed := writer.pingAt.Sub(started); elapsed < 15*time.Second || elapsed > 15*time.Second+250*time.Millisecond {
				t.Fatalf("heartbeat after %v", elapsed)
			}
		})
	}
}

type heartbeatRecorder struct {
	cancel  context.CancelFunc
	pings   int
	flushes int
	pingAt  time.Time
}

func (w *heartbeatRecorder) Write(p []byte) (int, error) {
	if string(p) == "{\"type\":\"ping\"}\n" {
		w.pings++
		w.pingAt = time.Now()
		w.cancel()
	}
	return len(p), nil
}

func (w *heartbeatRecorder) Flush() {
	w.flushes++
}

func useShortHeartbeat(t *testing.T) {
	t.Helper()
	previous := heartbeatInterval
	heartbeatInterval = 800 * time.Millisecond
	t.Cleanup(func() { heartbeatInterval = previous })
}

func TestCloudWatchRepeatedTimeoutsEndStream(t *testing.T) {
	useShortHeartbeat(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_MAX_ATTEMPTS", "1")

	var mu sync.Mutex
	var calls int
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var buf bytes.Buffer
	err := StreamCloudWatchLogToNDJSON(ctx, &buf, nil, "tasks", "task-1", "us-east-1", nil)
	if !errors.Is(err, errLogServiceTimeout) {
		t.Fatalf("stream error: %v", err)
	}
	if !strings.Contains(buf.String(), `"type":"error"`) || !strings.Contains(buf.String(), errLogServiceTimeout.Error()) {
		t.Fatalf("output = %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"type":"ping"`) {
		t.Fatalf("expected a heartbeat before the timeout error: %s", buf.String())
	}
	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls != maxConsecutiveLogTimeouts {
		t.Fatalf("GetLogEvents calls = %d, want %d", gotCalls, maxConsecutiveLogTimeouts)
	}
}

func TestCloudWatchTimeoutStreakResetsAfterLogPage(t *testing.T) {
	useShortHeartbeat(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_MAX_ATTEMPTS", "1")

	var mu sync.Mutex
	var calls int
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		call := calls
		mu.Unlock()
		if call == maxConsecutiveLogTimeouts || call == maxConsecutiveLogTimeouts*2 {
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			message := "first page"
			if call == maxConsecutiveLogTimeouts*2 {
				message = "second page"
			}
			_, _ = w.Write([]byte(`{"events":[{"message":"` + message + `","timestamp":1}],"nextForwardToken":"tok"}`))
			return
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	writer := &cancelAfterText{buf: &bytes.Buffer{}, cancel: cancel, needle: "second page"}
	err := StreamCloudWatchLogToNDJSON(ctx, writer, nil, "tasks", "task-1", "us-east-1", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stream error: %v output=%s", err, writer.buf.String())
	}
	if !strings.Contains(writer.buf.String(), "first page") || !strings.Contains(writer.buf.String(), "second page") {
		t.Fatalf("output = %s", writer.buf.String())
	}
	if strings.Contains(writer.buf.String(), `"type":"error"`) {
		t.Fatalf("successful page did not reset the timeout streak: %s", writer.buf.String())
	}
}

type cancelAfterText struct {
	buf    *bytes.Buffer
	cancel context.CancelFunc
	needle string
}

func (c *cancelAfterText) Write(p []byte) (int, error) {
	n, err := c.buf.Write(p)
	if strings.Contains(c.buf.String(), c.needle) {
		c.cancel()
	}
	return n, err
}
