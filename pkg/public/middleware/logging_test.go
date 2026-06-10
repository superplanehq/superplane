package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestRedactPathIDs(t *testing.T) {
	t.Run("redacts UUIDs in paths", func(t *testing.T) {
		assert.Equal(t,
			"/api/v1/integrations/{id}/webhook",
			redactPathIDs("/api/v1/integrations/e931a088-90c5-4fa2-9b30-fc7d9d586723/webhook"),
		)
	})

	t.Run("redacts multiple UUIDs", func(t *testing.T) {
		assert.Equal(t,
			"/api/v1/canvases/{id}/nodes/{id}",
			redactPathIDs("/api/v1/canvases/8f5fbc57-2738-409a-a6f8-af65c2de733c/nodes/0a1b2c3d-4e5f-6789-abcd-ef0123456789"),
		)
	})

	t.Run("leaves paths without UUIDs unchanged", func(t *testing.T) {
		assert.Equal(t, "/api/v1/healthz", redactPathIDs("/api/v1/healthz"))
	})

	t.Run("matches uppercase UUIDs", func(t *testing.T) {
		assert.Equal(t,
			"/api/v1/integrations/{id}/webhook",
			redactPathIDs("/api/v1/integrations/E931A088-90C5-4FA2-9B30-FC7D9D586723/webhook"),
		)
	})
}

func TestGroupingPathForRequest(t *testing.T) {
	t.Run("redacts UUIDs in path so all integration webhook errors group together", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/e931a088-90c5-4fa2-9b30-fc7d9d586723/webhook", nil)

		assert.Equal(t, "/api/v1/integrations/{id}/webhook", groupingPathForRequest(req))
	})

	t.Run("leaves paths without UUIDs unchanged", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)

		assert.Equal(t, "/api/v1/healthz", groupingPathForRequest(req))
	})
}
