package public

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
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
		r.Encryptor, r.Registry, signer, support.NewOIDCProvider(),
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

func Test__applySentryWebhookErrorTags_setsTagsOnEvent(t *testing.T) {
	transport := &captureTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              "https://examplePublicKey@o0.ingest.sentry.io/0",
		Transport:        transport,
		AttachStacktrace: false,
	})
	require.NoError(t, err)
	defer client.Flush(0)

	hub := sentry.NewHub(client, sentry.NewScope())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sentry/app/webhook", nil)
	req.Header.Set("Sentry-Hook-Resource", "issue")

	hub.WithScope(func(scope *sentry.Scope) {
		applySentryWebhookErrorTags(scope, req, []string{
			"installation_uuid", "inst-abc",
			"integration_id", "int-123",
			"inner_status", "500",
		})
		hub.CaptureException(errors.New("test error"))
	})
	hub.Flush(0)

	require.NotNil(t, transport.lastEvent)
	assert.Equal(t, "inst-abc", transport.lastEvent.Tags["installation_uuid"])
	assert.Equal(t, "int-123", transport.lastEvent.Tags["integration_id"])
	assert.Equal(t, "500", transport.lastEvent.Tags["inner_status"])
	assert.Equal(t, "issue", transport.lastEvent.Tags["hook_resource"])
}

func Test__applySentryWebhookErrorTags_noRequestAddsOnlyManualTags(t *testing.T) {
	transport := &captureTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              "https://examplePublicKey@o0.ingest.sentry.io/0",
		Transport:        transport,
		AttachStacktrace: false,
	})
	require.NoError(t, err)
	defer client.Flush(0)

	hub := sentry.NewHub(client, sentry.NewScope())

	hub.WithScope(func(scope *sentry.Scope) {
		applySentryWebhookErrorTags(scope, nil, []string{"installation_uuid", "inst-abc"})
		hub.CaptureException(errors.New("test error"))
	})
	hub.Flush(0)

	require.NotNil(t, transport.lastEvent)
	assert.Equal(t, "inst-abc", transport.lastEvent.Tags["installation_uuid"])
	_, hasHookResource := transport.lastEvent.Tags["hook_resource"]
	assert.False(t, hasHookResource)
}

func TestHandlerSentryAppWebhook_lookupFailure(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")

	r := support.Setup(t)
	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(
		r.Encryptor, r.Registry, signer, support.NewOIDCProvider(),
		"", "", "", "test", "/app/templates", r.AuthService, nil, false,
	)
	require.NoError(t, err)

	body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`)
	req := sentryWebhookRequest(body, "issue")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	server.HandleSentryAppWebhook(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerSentryAppWebhook_droppedDeliveryPreservesStatus(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")

	r := support.Setup(t)
	original := r.Registry.Integrations["sentry"]
	require.NotNil(t, original)
	r.Registry.Integrations["sentry"] = sentryWebhookStatusStub{
		Integration: original,
		status:      http.StatusBadGateway,
	}
	t.Cleanup(func() {
		r.Registry.Integrations["sentry"] = original
	})

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(
		r.Encryptor, r.Registry, signer, support.NewOIDCProvider(),
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

	transport := bindTestSentryHub(t)
	body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`)
	rec := httptest.NewRecorder()

	server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	events := transport.Events()
	delivery := requireEventWithTag(t, events, "inner_status", "502")
	assert.Equal(t, integration.ID.String(), delivery.Tags["integration_id"])
	assert.Equal(t, "issue", delivery.Tags["hook_resource"])
	assert.Contains(t, capturedExceptionText(delivery), "delivery failed for integration "+integration.ID.String())

	aggregate := requireEventWithTag(t, events, "inner_statuses", "502")
	assert.Equal(t, "install-1", aggregate.Tags["installation_uuid"])
	assert.Equal(t, integration.ID.String(), aggregate.Tags["integration_ids"])
	assert.Equal(t, "issue", aggregate.Tags["hook_resource"])
	assert.Contains(t, capturedExceptionText(aggregate), "dropped webhook delivery for installation install-1")
}

type sentryWebhookStatusStub struct {
	core.Integration
	status int
}

func (s sentryWebhookStatusStub) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(s.status)
}

func requireEventWithTag(t *testing.T, events []*sentry.Event, key, value string) *sentry.Event {
	t.Helper()
	for _, event := range events {
		if event != nil && event.Tags[key] == value {
			return event
		}
	}
	require.Fail(t, "expected a captured event with tag", "%s=%s events=%d", key, value, len(events))
	return nil
}

func Test__joinStatuses(t *testing.T) {
	assert.Equal(t, "", joinStatuses(nil))
	assert.Equal(t, "500", joinStatuses([]int{500}))
	assert.Equal(t, "502,503", joinStatuses([]int{502, 503}))
}

type captureTransport struct {
	lastEvent *sentry.Event
}

func (t *captureTransport) Configure(options sentry.ClientOptions) {}

func (t *captureTransport) SendEvent(event *sentry.Event) {
	t.lastEvent = event
}

func (t *captureTransport) Flush(timeout time.Duration) bool {
	return true
}
