package public

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
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

func captureSentryWebhookLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	logger := logging.SentryWebhookLogger()
	previous := logger.Out
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})
	return buffer
}

func sentryWebhookLogLine(t *testing.T, raw string, message string) map[string]any {
	t.Helper()

	for _, line := range bytes.Split([]byte(raw), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(line, &payload))
		if payload["message"] == message {
			return payload
		}
	}
	t.Fatalf("missing Sentry webhook log %q", message)
	return nil
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

func TestHandleSentryAppWebhook_canceledCommitReturnsServerError(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")

	r := support.Setup(t)
	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(
		r.Encryptor, r.Registry, signer, support.NewOIDCProvider(),
		"", "", "", "test", "/app/templates", r.AuthService, false,
	)
	require.NoError(t, err)

	body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`)
	req := signedSentryRequest("/api/v1/sentry/app/webhook", body, "csecret")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	server.HandleSentryAppWebhook(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
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
