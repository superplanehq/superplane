package slack

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/integrations/httpx"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func withDefaultTransport(t *testing.T, rt roundTripFunc) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = original
	})
}

// withFastRetries swaps slackRetryConfig to delays small enough
// that retry tests stay sub-second. Restored on test cleanup.
func withFastRetries(t *testing.T, attempts int) {
	t.Helper()
	original := slackRetryConfig
	slackRetryConfig = httpx.Config{
		MaxAttempts: attempts,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    5 * time.Millisecond,
	}
	t.Cleanup(func() {
		slackRetryConfig = original
	})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}
