package middleware

import (
	"net/http"
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
