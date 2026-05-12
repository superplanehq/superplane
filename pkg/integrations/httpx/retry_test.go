package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__Do__SucceedsAfterTransientFailures(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(context.Background(), server.Client(), req, fastConfig(3))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}

func Test__Do__ReturnsLastResponseWhenAllAttemptsFail(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(context.Background(), server.Client(), req, fastConfig(3))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func Test__Do__DoesNotRetryOn4xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(context.Background(), server.Client(), req, fastConfig(3))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "4xx must not retry")
}

func Test__Do__HonorsRetryAfterHeader(t *testing.T) {
	var attempts int32
	var firstAttempt time.Time
	var secondAttempt time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		switch n {
		case 1:
			firstAttempt = time.Now()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			secondAttempt = time.Now()
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	cfg := Config{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Second}
	resp, err := Do(context.Background(), server.Client(), req, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	gap := secondAttempt.Sub(firstAttempt)
	assert.GreaterOrEqual(t, gap, 900*time.Millisecond, "Retry-After: 1 should produce ~1s gap, got %s", gap)
}

func Test__Do__AbortsOnContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	cfg := Config{MaxAttempts: 10, BaseDelay: 200 * time.Millisecond, MaxDelay: 5 * time.Second}
	_, err = Do(ctx, server.Client(), req, cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, got %v", err)
}

func Test__Do__ReplaysBodyOnRetry(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		if len(bodies) < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("payload"))
	require.NoError(t, err)

	resp, err := Do(context.Background(), server.Client(), req, fastConfig(2))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, bodies, 2)
	assert.Equal(t, "payload", bodies[0])
	assert.Equal(t, "payload", bodies[1])
}

func Test__Do__RetriesOnNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Capture URL then close the server so the first call yields a
	// connection error. We then re-open a server at the same port —
	// but that's flaky cross-platform. Instead, drive the retry
	// path through a custom transport that fails the first call.
	url := server.URL
	server.Close()

	var attempts int32
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			return nil, errors.New("simulated dial error")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	})
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := Do(context.Background(), client, req, fastConfig(3))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

func Test__Do__RejectsBodyWithoutGetBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	// http.NewRequest sets GetBody for *strings.Reader; bypass that
	// by attaching the body manually.
	req, err := http.NewRequest(http.MethodPost, server.URL, nil)
	require.NoError(t, err)
	req.Body = io.NopCloser(strings.NewReader("payload"))
	req.GetBody = nil

	_, err = Do(context.Background(), server.Client(), req, fastConfig(2))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GetBody")
}

func Test__ParseRetryAfter(t *testing.T) {
	t.Run("empty -> 0", func(t *testing.T) {
		assert.Zero(t, parseRetryAfter(""))
	})
	t.Run("seconds", func(t *testing.T) {
		assert.Equal(t, 5*time.Second, parseRetryAfter("5"))
	})
	t.Run("zero seconds -> 0", func(t *testing.T) {
		assert.Zero(t, parseRetryAfter("0"))
	})
	t.Run("http-date in future", func(t *testing.T) {
		future := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
		got := parseRetryAfter(future)
		assert.Greater(t, got, 500*time.Millisecond)
		assert.LessOrEqual(t, got, 2*time.Second)
	})
	t.Run("http-date in past -> 0", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour).UTC().Format(http.TimeFormat)
		assert.Zero(t, parseRetryAfter(past))
	})
	t.Run("garbage -> 0", func(t *testing.T) {
		assert.Zero(t, parseRetryAfter("not-a-thing"))
	})
}

func Test__BackoffDelay__BoundsAreRespected(t *testing.T) {
	for attempt := 1; attempt <= 8; attempt++ {
		got := backoffDelay(attempt, 100*time.Millisecond, 2*time.Second)
		assert.LessOrEqual(t, got, 2*time.Second, fmt.Sprintf("attempt=%d", attempt))
		assert.GreaterOrEqual(t, got, time.Duration(0), fmt.Sprintf("attempt=%d", attempt))
	}
}

func fastConfig(attempts int) Config {
	return Config{
		MaxAttempts: attempts,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		RetryOn:     DefaultRetryOn,
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
