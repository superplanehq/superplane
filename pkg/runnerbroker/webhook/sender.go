package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/superplanehq/superplane/pkg/runnerbroker/api"
)

// Sender delivers webhook callbacks with bounded retries.
type Sender struct {
	Client  *http.Client
	Retries int
	Log     *slog.Logger // optional: logs webhook_delivery per attempt
}

// DefaultSender uses a sensible HTTP client and a few retries.
func DefaultSender() *Sender {
	return &Sender{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Retries: 3,
	}
}

func webhookHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

// Deliver POSTs JSON to url until success or retries exhausted.
func (s *Sender) Deliver(ctx context.Context, webhookURL string, payload api.WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var last error
	attempts := s.Retries
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff(i)):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "superplane/webhook")

		start := time.Now()
		resp, err := s.Client.Do(req)
		dur := time.Since(start)
		attemptNum := i + 1

		base := []any{
			slog.Int("attempt", attemptNum),
			slog.String("task_id", payload.TaskID),
			slog.String("status_outcome", payload.Status),
			slog.String("url_host", webhookHost(webhookURL)),
			slog.Duration("dur", dur),
		}

		if err != nil {
			last = err
			if s.Log != nil {
				s.Log.Warn("webhook_delivery", append(base, slog.Any("err", err))...)
			}
			continue
		}
		code := resp.StatusCode
		_ = resp.Body.Close()

		switch {
		case code >= 200 && code < 300:
			if s.Log != nil {
				s.Log.Info("webhook_delivery", append(base, slog.Int("http_status", code))...)
			}
			return nil
		default:
			last = fmt.Errorf("webhook status %d", code)
			if s.Log != nil {
				s.Log.Warn("webhook_delivery", append(base, slog.Int("http_status", code))...)
			}
		}
	}
	if last != nil && s.Log != nil {
		s.Log.Error("webhook_delivery_exhausted",
			slog.Int("attempts", attempts),
			slog.String("task_id", payload.TaskID),
			slog.String("url_host", webhookHost(webhookURL)),
			slog.Any("last_err", last),
		)
	}
	if last != nil {
		return fmt.Errorf("webhook failed after %d attempts: %w", attempts, last)
	}
	return fmt.Errorf("webhook failed after %d attempts", attempts)
}

func backoff(attempt int) time.Duration {
	const maxShift = 6
	if attempt > maxShift {
		attempt = maxShift
	}
	d := time.Duration(1<<attempt) * 100 * time.Millisecond
	if d > 10*time.Second {
		return 10 * time.Second
	}
	return d
}
