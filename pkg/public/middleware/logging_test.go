package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
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

func TestLoggingMiddleware_GeneratesAndEchoesRequestID(t *testing.T) {
	logger, buffer := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, w.Header().Get(requestIDHeader), RequestIDFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/devzero-inc", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.8, 35.191.62.152")
	request.Header.Set("X-Cloud-Trace-Context", "105445aa7843bc8bf206b12000100000/1;o=1")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	requestID := recorder.Header().Get(requestIDHeader)
	_, err := uuid.Parse(requestID)
	require.NoError(t, err)

	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, "info", payload["level"])
	assert.Equal(t, "handled request /api/v1/organizations/devzero-inc", payload["msg"])
	assert.Equal(t, "/api/v1/organizations/devzero-inc", payload["path"])
	assert.Equal(t, requestID, payload["request_id"])
	assert.Equal(t, "35.191.62.152", payload["client_ip"])
	assert.Equal(t, "105445aa7843bc8bf206b12000100000", payload["trace_id"])
	assert.EqualValues(t, http.StatusOK, payload["status"])
	_, hasDuration := payload["duration_ms"]
	assert.True(t, hasDuration)
	_, hasDurationNanos := payload["duration"]
	assert.False(t, hasDurationNanos)
}

func TestLoggingMiddleware_ClientIPPrefersProxyHeader(t *testing.T) {
	logger, buffer := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/devzero-inc", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.8, 198.51.100.4")
	request.Header.Set("CF-Connecting-IP", "198.51.100.20")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, "198.51.100.20", payload["client_ip"])
}

func TestRequestLogAccountIdentity_UsesEffectiveAccount(t *testing.T) {
	admin := &models.Account{ID: uuid.New()}
	effective := &models.Account{ID: uuid.New()}

	logged, adminID := requestLogAccountIdentity(admin, effective, admin.ID.String())
	assert.Equal(t, effective.ID, logged.ID)
	assert.Equal(t, admin.ID.String(), adminID)

	logged, adminID = requestLogAccountIdentity(admin, nil, admin.ID.String())
	assert.Equal(t, admin.ID, logged.ID)
	assert.Empty(t, adminID)
}

func TestLoggingMiddleware_KeepsIncomingRequestID(t *testing.T) {
	logger, buffer := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/devzero-inc", nil)
	request.Header.Set(requestIDHeader, "req-123")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, "req-123", recorder.Header().Get(requestIDHeader))
	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, "req-123", payload["request_id"])
	assert.Equal(t, "info", payload["level"])
}

func TestLoggingMiddleware_ReplacesUnsafeRequestID(t *testing.T) {
	logger, buffer := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/devzero-inc", nil)
	request.Header.Set(requestIDHeader, "bad\nid")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	requestID := recorder.Header().Get(requestIDHeader)
	assert.NotContains(t, requestID, "\n")
	_, err := uuid.Parse(requestID)
	require.NoError(t, err)
	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, requestID, payload["request_id"])
}

func TestLoggingMiddleware_LogsIdentityFromRequest(t *testing.T) {
	logger, buffer := newJSONLogger()
	organizationID := uuid.New()
	userID := uuid.New()
	accountID := uuid.New()
	user := &models.User{
		ID:             userID,
		OrganizationID: organizationID,
		AccountID:      &accountID,
	}

	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRequestLogUser(r.Context(), user)
		SetRequestLogImpersonator(r.Context(), "admin-account")
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/devzero-inc/integrations", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, organizationID.String(), payload["organization_id"])
	assert.Equal(t, userID.String(), payload["user_id"])
	assert.Equal(t, accountID.String(), payload["account_id"])
	assert.Equal(t, "admin-account", payload["impersonator_account_id"])
}

func TestLoggingMiddleware_ServerErrorUsesErrorLevel(t *testing.T) {
	logger, buffer := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/devzero-inc", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	payload := decodeLogLine(t, buffer.String())
	assert.Equal(t, "error", payload["level"])
	assert.Equal(t, "handled request /api/v1/organizations/devzero-inc", payload["msg"])
	assert.EqualValues(t, http.StatusInternalServerError, payload["status"])
}

func TestLoggingMiddleware_ServerErrorWithoutCauseKeepsStatusMessage(t *testing.T) {
	transport := bindMiddlewareSentryHub(t)
	logger, _ := newJSONLogger()
	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/missing", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	events := transport.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "HTTP 502 /api/v1/webhooks/missing", events[0].Message)
	assert.Empty(t, events[0].Exception)
	assert.Equal(t, "502", events[0].Tags["status"])
}

func bindMiddlewareSentryHub(t *testing.T) *memorySentryTransport {
	t.Helper()
	transport := &memorySentryTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://public@localhost/1",
		Transport: transport,
	})
	require.NoError(t, err)
	hub := sentry.CurrentHub()
	previous := hub.Client()
	hub.BindClient(client)
	t.Cleanup(func() {
		hub.BindClient(previous)
	})
	return transport
}

type memorySentryTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *memorySentryTransport) Configure(sentry.ClientOptions) {}

func (t *memorySentryTransport) Flush(time.Duration) bool { return true }

func (t *memorySentryTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	clone := *event
	t.events = append(t.events, &clone)
}

func (t *memorySentryTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

func newJSONLogger() (*log.Logger, *bytes.Buffer) {
	logger := log.New()
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	logger.SetFormatter(&log.JSONFormatter{})
	return logger, buffer
}

func decodeLogLine(t *testing.T, raw string) map[string]any {
	t.Helper()

	payload := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	return payload
}
