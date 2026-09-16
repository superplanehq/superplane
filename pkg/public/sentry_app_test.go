package public

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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

func TestHandleSentryAppWebhook_answersSentryWithTheDeliveryResult(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")

	r := support.Setup(t)
	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(
		r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), r.GitProvider,
		"", "", "", "test", "/app/templates", r.AuthService, nil, false,
	)
	require.NoError(t, err)

	integration, err := models.CreateIntegration(
		uuid.New(), r.Organization.ID, "sentry", support.RandomName("sentry"), map[string]any{},
	)
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"hostedApp":        true,
		"installationUUID": "install-1",
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	listening, err := models.ListSentryIntegrationsByInstallationUUID(database.Conn(), "install-1")
	require.NoError(t, err)
	require.Len(t, listening, 1)

	t.Run("a delivered event is accepted", func(t *testing.T) {
		body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`)
		rec := httptest.NewRecorder()

		server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("a rejected event is accepted, because Sentry cannot fix it", func(t *testing.T) {
		body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":"not-an-object"}`)
		rec := httptest.NewRecorder()

		server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func Test__sentryDeliveryFinished(t *testing.T) {
	assert.True(t, sentryDeliveryFinished(http.StatusOK))
	assert.True(t, sentryDeliveryFinished(http.StatusForbidden))
	assert.False(t, sentryDeliveryFinished(http.StatusInternalServerError))
	assert.False(t, sentryDeliveryFinished(http.StatusBadGateway))
}

func sentryWebhookRequest(body []byte, resource string) *http.Request {
	mac := hmac.New(sha256.New, []byte("csecret"))
	mac.Write(body)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/sentry/app/webhook", bytes.NewReader(body))
	request.Header.Set("Sentry-Hook-Signature", hex.EncodeToString(mac.Sum(nil)))
	request.Header.Set("Sentry-Hook-Resource", resource)
	return request
}

func Test__isHostedSentryAppBrowserCallback(t *testing.T) {
	hosted := &models.Integration{
		AppName: "sentry",
		Metadata: datatypes.NewJSONType(map[string]any{
			"hostedApp": true,
		}),
	}
	legacy := &models.Integration{
		AppName: "sentry",
		Metadata: datatypes.NewJSONType(map[string]any{
			"hostedApp": false,
		}),
	}

	setup := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/"+uuid.NewString()+"/setup", nil)
	install := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/"+uuid.NewString()+"/install", nil)
	webhook := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/"+uuid.NewString()+"/webhook", nil)

	assert.True(t, isHostedSentryAppBrowserCallback(setup, hosted))
	assert.True(t, isHostedSentryAppBrowserCallback(install, hosted))
	assert.False(t, isHostedSentryAppBrowserCallback(setup, legacy))
	assert.False(t, isHostedSentryAppBrowserCallback(webhook, hosted))
}
