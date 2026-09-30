package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/telemetry"
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

func TestLoggingMiddlewareErrorReporting(t *testing.T) {
	t.Run("reports the error recorded behind the response", func(t *testing.T) {
		transport := captureSentryEvents(t)

		serveWithLoggingMiddleware(t, "/api/v1/factories", func(w http.ResponseWriter, r *http.Request) {
			telemetry.RecordServerError(r.Context(), errors.New(`ERROR: column factories.key does not exist (SQLSTATE 42703)`))
			w.WriteHeader(http.StatusInternalServerError)
		})

		require.Len(t, transport.events, 1)
		event := transport.events[0]

		require.Len(t, event.Exception, 1)
		assert.Equal(t, `ERROR: column factories.key does not exist (SQLSTATE 42703)`, event.Exception[0].Value)
		assert.Equal(t, "500", event.Tags["status"])
		assert.Empty(t, event.Message)
	})

	t.Run("falls back to the request summary when no error was recorded", func(t *testing.T) {
		transport := captureSentryEvents(t)

		serveWithLoggingMiddleware(t, "/api/v1/factories", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		require.Len(t, transport.events, 1)
		event := transport.events[0]

		assert.Equal(t, "HTTP 500 /api/v1/factories", event.Message)
		assert.Empty(t, event.Exception)
	})

	t.Run("reports nothing for successful responses", func(t *testing.T) {
		transport := captureSentryEvents(t)

		serveWithLoggingMiddleware(t, "/api/v1/factories", func(w http.ResponseWriter, r *http.Request) {
			telemetry.RecordServerError(r.Context(), errors.New("recorded but recovered from"))
			w.WriteHeader(http.StatusOK)
		})

		assert.Empty(t, transport.events)
	})
}

func serveWithLoggingMiddleware(t *testing.T, path string, handler http.HandlerFunc) {
	t.Helper()

	logger := log.New()
	logger.SetOutput(io.Discard)

	request := httptest.NewRequest(http.MethodGet, path, nil)
	LoggingMiddleware(logger)(handler).ServeHTTP(httptest.NewRecorder(), request)
}

// captureSentryEvents binds a client with an in-memory transport to the current
// hub for the duration of the test.
func captureSentryEvents(t *testing.T) *sentryTransportStub {
	t.Helper()

	transport := &sentryTransportStub{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	})
	require.NoError(t, err)

	hub := sentry.CurrentHub()
	previous := hub.Client()
	hub.BindClient(client)
	t.Cleanup(func() { hub.BindClient(previous) })

	return transport
}

type sentryTransportStub struct {
	events []*sentry.Event
}

func (t *sentryTransportStub) Configure(sentry.ClientOptions) {}

func (t *sentryTransportStub) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}

func (t *sentryTransportStub) Flush(time.Duration) bool { return true }
