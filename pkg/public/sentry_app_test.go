package public

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/config"
)

func TestHandleSentryAppInstall_missingState(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sentry/app/install", nil)
	rec := httptest.NewRecorder()

	server.HandleSentryAppInstall(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleSentryAppWebhook_notConfigured(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "")
	t.Setenv(config.EnvSentryAppClientID, "")
	t.Setenv(config.EnvSentryAppClientSecret, "")

	server := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sentry/app/webhook", nil)
	rec := httptest.NewRecorder()

	server.HandleSentryAppWebhook(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
