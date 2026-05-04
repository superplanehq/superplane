package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/superplane/runner/shared/api"
)

// Sender delivers webhook callbacks with bounded retries.
type Sender struct {
	Client  *http.Client
	Retries int
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

// Deliver POSTs JSON to url until success or retries exhausted.
func (s *Sender) Deliver(ctx context.Context, url string, payload api.WebhookPayload) error {
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
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "superplane-fleet-manager/webhook")

		resp, err := s.Client.Do(req)
		if err != nil {
			last = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		last = fmt.Errorf("webhook status %d", resp.StatusCode)
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
