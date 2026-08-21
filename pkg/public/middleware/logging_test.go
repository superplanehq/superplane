package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/requesterror"
)

func TestShouldCaptureHTTPError(t *testing.T) {
	t.Run("captures real 5xx server errors", func(t *testing.T) {
		assert.True(t, shouldCaptureHTTPError(http.StatusInternalServerError))
		assert.True(t, shouldCaptureHTTPError(http.StatusBadGateway))
		assert.True(t, shouldCaptureHTTPError(http.StatusServiceUnavailable))
		assert.True(t, shouldCaptureHTTPError(http.StatusGatewayTimeout))
	})

	t.Run("skips non-5xx responses", func(t *testing.T) {
		assert.False(t, shouldCaptureHTTPError(http.StatusOK))
		assert.False(t, shouldCaptureHTTPError(http.StatusNotFound))
		assert.False(t, shouldCaptureHTTPError(http.StatusUnauthorized))
		assert.False(t, shouldCaptureHTTPError(http.StatusTeapot))
		assert.False(t, shouldCaptureHTTPError(499))
	})

	t.Run("skips 501 Not Implemented", func(t *testing.T) {
		// grpc-gateway responds with 501 when a route exists but the HTTP
		// method has no mapping (e.g. POST /api/v1/triggers/start). That is a
		// client-caused error, not a server bug, so we should not spam Sentry.
		assert.False(t, shouldCaptureHTTPError(http.StatusNotImplemented))
	})

	t.Run("skips 505 HTTP Version Not Supported", func(t *testing.T) {
		assert.False(t, shouldCaptureHTTPError(http.StatusHTTPVersionNotSupported))
	})
}

func TestLoggingMiddlewareCapturesServerErrors(t *testing.T) {
	t.Run("sends the recorded cause as an exception", func(t *testing.T) {
		transport := bindSentryTestClient(t)
		cause := errors.New("column factories.next_work_order_number does not exist")

		serveWithLoggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			requesterror.Record(r.Context(), cause)
			w.WriteHeader(http.StatusInternalServerError)
		})

		events := transport.events()
		require.Len(t, events, 1)
		require.Len(t, events[0].Exception, 1)
		assert.Equal(t, cause.Error(), events[0].Exception[0].Value)
		assert.Equal(t, "500", events[0].Tags["status"])
	})

	t.Run("sends a message when no cause is recorded", func(t *testing.T) {
		transport := bindSentryTestClient(t)

		serveWithLoggingMiddleware(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		events := transport.events()
		require.Len(t, events, 1)
		assert.Empty(t, events[0].Exception)
		assert.Equal(t, "HTTP 500 /api/v1/factories", events[0].Message)
	})

	t.Run("sends nothing for a successful request", func(t *testing.T) {
		transport := bindSentryTestClient(t)

		serveWithLoggingMiddleware(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		assert.Empty(t, transport.events())
	})
}

func serveWithLoggingMiddleware(handler http.HandlerFunc) {
	logger := log.New()
	logger.SetOutput(io.Discard)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/factories", nil)
	LoggingMiddleware(logger)(handler).ServeHTTP(httptest.NewRecorder(), r)
}

// bindSentryTestClient binds a Sentry client that keeps the events in memory,
// and restores the previous client when the test ends.
func bindSentryTestClient(t *testing.T) *sentryTestTransport {
	t.Helper()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:        "https://public@example.com/1",
		Transport:  transport,
		SampleRate: 1.0,
	})
	require.NoError(t, err)

	hub := sentry.CurrentHub()
	previous := hub.Client()
	hub.BindClient(client)
	t.Cleanup(func() { hub.BindClient(previous) })

	return transport
}

type sentryTestTransport struct {
	mu       sync.Mutex
	captured []*sentry.Event
}

func (t *sentryTestTransport) Configure(_ sentry.ClientOptions) {}

func (t *sentryTestTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.captured = append(t.captured, event)
}

func (t *sentryTestTransport) Flush(_ time.Duration) bool { return true }

func (t *sentryTestTransport) events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.captured
}
