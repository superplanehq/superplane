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
	t.Run("redacts a single UUID", func(t *testing.T) {
		assert.Equal(t,
			"/api/v1/integrations/{id}/webhook",
			redactPathIDs("/api/v1/integrations/e931a088-90c5-4fa2-9b30-fc7d9d586723/webhook"),
		)
	})

	t.Run("redacts multiple UUIDs in the same path", func(t *testing.T) {
		assert.Equal(t,
			"/api/v1/canvases/{id}/executions/{id}/hooks/approve",
			redactPathIDs("/api/v1/canvases/07c03f78-499d-4324-a1b3-26936e61a831/executions/eb157a63-4664-4dc4-aacf-f09924e95e17/hooks/approve"),
		)
	})

	t.Run("redacts uppercase UUIDs", func(t *testing.T) {
		assert.Equal(t,
			"/api/v1/canvases/{id}/foo",
			redactPathIDs("/api/v1/canvases/07C03F78-499D-4324-A1B3-26936E61A831/foo"),
		)
	})

	t.Run("leaves paths without UUIDs alone", func(t *testing.T) {
		assert.Equal(t,
			"/api/v1/canvases",
			redactPathIDs("/api/v1/canvases"),
		)
	})

	t.Run("does not redact short hex-looking segments", func(t *testing.T) {
		// numeric/short ids are kept as-is so we still group meaningful paths
		assert.Equal(t,
			"/api/v1/runs/1234",
			redactPathIDs("/api/v1/runs/1234"),
		)
	})
}

func TestGroupingPathForRequest(t *testing.T) {
	t.Run("redacts UUIDs from the request URL path", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/canvases/07c03f78-499d-4324-a1b3-26936e61a831/executions/eb157a63-4664-4dc4-aacf-f09924e95e17/hooks/approve", nil)
		assert.Equal(t,
			"/api/v1/canvases/{id}/executions/{id}/hooks/approve",
			groupingPathForRequest(r),
		)
	})
}
