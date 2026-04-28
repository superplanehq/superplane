// Package httpx provides small, dependency-free HTTP helpers shared
// across integrations. The retry helper here exists so outbound
// integration calls survive transient 5xx / 429 / network errors
// without each integration hand-rolling its own retry loop.
package httpx

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Config tunes Do's retry behavior. Zero values are replaced with
// the defaults from DefaultConfig().
type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration

	// RetryOn decides whether (resp, err) should trigger another
	// attempt. If nil, DefaultRetryOn is used.
	RetryOn func(*http.Response, error) bool
}

// DefaultConfig returns the recommended defaults: 3 attempts,
// 200ms base delay with full jitter, capped at 5s.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		RetryOn:     DefaultRetryOn,
	}
}

// DefaultRetryOn retries on transport errors, 429, and 5xx.
func DefaultRetryOn(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp == nil {
		return false
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode >= 500 && resp.StatusCode <= 599
}

// Do executes req via client with retries per cfg. The supplied
// context is bound to every attempt and is also honored during
// backoff sleep — a cancellation aborts the loop with ctx.Err().
//
// If req has a body, req.GetBody must be set so the body can be
// re-read on each attempt. http.NewRequest sets GetBody
// automatically for *bytes.Buffer, *bytes.Reader, and
// *strings.Reader; callers using other readers must set it
// explicitly.
//
// On a retried response, the body is drained and closed before the
// next attempt to free the underlying connection. The final
// response (whether retryable or not) is returned to the caller
// undrained, matching net/http's contract.
func Do(ctx context.Context, client *http.Client, req *http.Request, cfg Config) (*http.Response, error) {
	if client == nil {
		return nil, errors.New("httpx: nil client")
	}
	if req == nil {
		return nil, errors.New("httpx: nil request")
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 200 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 5 * time.Second
	}
	if cfg.RetryOn == nil {
		cfg.RetryOn = DefaultRetryOn
	}

	for attempt := 1; ; attempt++ {
		attemptReq := req.Clone(ctx)
		if req.Body != nil {
			if req.GetBody == nil {
				return nil, errors.New("httpx: request has body but GetBody is unset; retries impossible")
			}
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attemptReq.Body = body
		}

		resp, err := client.Do(attemptReq)

		if !cfg.RetryOn(resp, err) || attempt >= cfg.MaxAttempts {
			return resp, err
		}

		// Drain & close before retrying so the connection can be reused.
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		delay := backoffDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)
		if resp != nil {
			if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
				if ra > cfg.MaxDelay {
					ra = cfg.MaxDelay
				}
				delay = ra
			}
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// backoffDelay returns an exponential backoff with full jitter,
// capped at maxDelay. Attempt is 1-indexed.
func backoffDelay(attempt int, base, maxDelay time.Duration) time.Duration {
	shift := attempt - 1
	if shift > 30 {
		shift = 30
	}
	upper := base << shift
	if upper <= 0 || upper > maxDelay {
		upper = maxDelay
	}
	if upper <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(upper)))
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if secs, err := strconv.Atoi(value); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}
