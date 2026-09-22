package public

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteWebhookError(t *testing.T) {
	webhookID := uuid.New()

	t.Run("keeps client errors visible", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeWebhookError(rec, webhookID, http.StatusBadRequest, errors.New("signature invalid"))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "signature invalid")
	})

	t.Run("hides internal errors", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeWebhookError(rec, webhookID, http.StatusInternalServerError, errors.New("db exploded"))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "error handling webhook\n", rec.Body.String())
	})

	t.Run("maps canceled requests to 503", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeWebhookError(rec, webhookID, http.StatusInternalServerError, context.Canceled)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Equal(t, "error handling webhook\n", rec.Body.String())
	})
}

func TestWriteWebhookErrorRequiresStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	writeWebhookError(rec, uuid.New(), 0, errors.New("unknown"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
