package public

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
		"", "", "", "test", "/app/templates", r.AuthService, false,
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
		logs := captureSentryWebhookLogs(t)

		server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

		assert.Equal(t, http.StatusOK, rec.Code)
		received := sentryWebhookLogLine(t, logs.String(), "Sentry app webhook received")
		payload, ok := received["payload"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "created", payload["action"])

		var receipts []models.SentryWebhookReceipt
		require.NoError(t, database.Conn().Where("installation_uuid = ? AND issue_id = ?", "install-1", "1").Find(&receipts).Error)
		require.NotEmpty(t, receipts)
		assert.Equal(t, models.SentryWebhookOutcomeAccepted, receipts[0].Outcome)
		assert.Equal(t, "issue", receipts[0].HookResource)
		assert.Equal(t, "created", receipts[0].Action)
	})

	t.Run("a rejected event is accepted, because Sentry cannot fix it", func(t *testing.T) {
		body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":"not-an-object"}`)
		rec := httptest.NewRecorder()

		server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

		assert.Equal(t, http.StatusOK, rec.Code)

		var receipts []models.SentryWebhookReceipt
		require.NoError(t, database.Conn().
			Where("installation_uuid = ? AND outcome = ?", "install-1", models.SentryWebhookOutcomeRejected).
			Find(&receipts).Error)
		require.NotEmpty(t, receipts)
	})

	t.Run("an invalid signature does not store a receipt", func(t *testing.T) {
		var before int64
		require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&before).Error)

		request := httptest.NewRequest(http.MethodPost, "/api/v1/sentry/app/webhook", bytes.NewReader([]byte(`{"action":"created","secret":"caller-chosen"}`)))
		request.Header.Set("Sentry-Hook-Signature", "not-a-signature")
		request.Header.Set("Sentry-Hook-Resource", "issue")
		rec := httptest.NewRecorder()
		logs := captureSentryWebhookLogs(t)

		server.HandleSentryAppWebhook(rec, request)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var after int64
		require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&after).Error)
		assert.Equal(t, before, after)

		rejected := sentryWebhookLogLine(t, logs.String(), "Sentry app webhook was rejected")
		_, hasPayload := rejected["payload"]
		assert.False(t, hasPayload)
		assert.Equal(t, "issue", rejected["hook_resource"])
	})
}

func TestHandleSentryAppWebhook_listenerCannotStartRun(t *testing.T) {
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

	integration, err := models.CreateIntegration(
		uuid.New(), r.Organization.ID, "sentry", support.RandomName("sentry"), map[string]any{},
	)
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"hostedApp":        true,
		"installationUUID": "install-1",
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`)

	t.Run("a listener whose canvas has no live version is acknowledged", func(t *testing.T) {
		canvas := createSentryIssueListener(t, r, integration)

		// The schema keeps live_version_id NOT NULL, so a canvas without a
		// live version cannot be stored. Point the canvas at the live version
		// of another canvas instead: the live version lookup for this canvas
		// then finds nothing, the same not-found error a nil live version
		// produces.
		other, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		require.NotNil(t, other.LiveVersionID)
		require.NoError(t, database.Conn().
			Model(&models.Canvas{}).
			Where("id = ?", canvas.ID).
			Update("live_version_id", *other.LiveVersionID).
			Error)

		rec := httptest.NewRecorder()
		server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("a listener whose canvas is gone is acknowledged", func(t *testing.T) {
		canvas := createSentryIssueListener(t, r, integration)
		require.NoError(t, database.Conn().Delete(&models.Canvas{}, canvas.ID).Error)

		rec := httptest.NewRecorder()
		server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleSentryAppWebhook_unclaimedInstallWithoutCodeLogsSentryOrganization(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")

	r := support.Setup(t)
	server, err := NewServer(
		r.Encryptor, r.Registry, jwt.NewSigner("test-client-secret"), support.NewOIDCProvider(),
		"", "", "", "test", "/app/templates", r.AuthService, false,
	)
	require.NoError(t, err)

	body := []byte(`{
		"action": "created",
		"installation": {"uuid": "unclaimed-install"},
		"data": {"installation": {
			"status": "installed",
			"uuid": "unclaimed-install",
			"organization": {"slug": "acme-sentry", "id": 127789}
		}}
	}`)
	rec := httptest.NewRecorder()
	logs := captureSentryWebhookLogs(t)

	server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "installation"))

	assert.Equal(t, http.StatusOK, rec.Code)
	failed := sentryWebhookLogLine(t, logs.String(), "failed to store unclaimed Sentry app install")
	assert.Equal(t, "unclaimed-install", failed["installation_uuid"])
	assert.Equal(t, "acme-sentry", failed["sentry_organization_slug"])
	assert.Equal(t, "127789", failed["sentry_organization_id"])
	assert.Equal(t, "installation grant code is required", failed["error"])
}

func createSentryIssueListener(t *testing.T, r *support.ResourceRegistry, integration *models.Integration) *models.Canvas {
	t.Helper()

	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        "listener",
				Name:          "listener",
				Type:          models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "sentry.onIssue"}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)

	_, err := models.CreateIntegrationSubscription(&nodes[0], integration, map[string]any{
		"resources": []string{"issue"},
	})
	require.NoError(t, err)

	return canvas
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

func TestHandlerSentryAppWebhook_lookupCanceledWhileRunning(t *testing.T) {
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

	transport := bindTestSentryHub(t)
	body := []byte(`{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := sentryWebhookRequest(body, "issue").WithContext(ctx)

	lookupStarted := make(chan struct{})
	const callback = "test:sentry-app-webhook-lookup-canceled"
	db := database.Conn()
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		holdSentryInstallationLookupUntilCanceled(tx, lookupStarted)
	}))
	t.Cleanup(func() {
		require.NoError(t, db.Callback().Query().Remove(callback))
	})

	var before int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&before).Error)

	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.HandleSentryAppWebhook(rec, req)
	}()

	select {
	case <-lookupStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("installation lookup did not start")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("webhook handler did not return after the caller disconnected")
	}

	assert.Equal(t, statusClientClosedRequest, rec.Code)
	assert.Empty(t, transport.Events())

	var after int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&after).Error)
	assert.Equal(t, before, after)
}

func holdSentryInstallationLookupUntilCanceled(tx *gorm.DB, started chan struct{}) {
	if tx == nil || tx.Statement == nil || tx.Statement.Table != "app_installations" {
		return
	}
	requestCtx := tx.Statement.Context
	if requestCtx == nil {
		return
	}

	select {
	case <-started:
	default:
		close(started)
	}
	<-requestCtx.Done()
	tx.AddError(requestCtx.Err())
}

func TestHandlerSentryAppWebhook_lookupErrorIsReported(t *testing.T) {
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

	transport := bindTestSentryHub(t)
	installationUUID := uuid.NewString()
	body := []byte(`{"action":"created","installation":{"uuid":"` + installationUUID + `"},"data":{"issue":{"id":"1"}}}`)

	db := database.Conn()
	const callback = "test:sentry-app-webhook-lookup-failure"
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil || tx.Statement.Table != "app_installations" {
			return
		}
		tx.AddError(errors.New("integrations unavailable"))
	}))
	t.Cleanup(func() {
		require.NoError(t, db.Callback().Query().Remove(callback))
	})

	rec := httptest.NewRecorder()
	server.HandleSentryAppWebhook(rec, sentryWebhookRequest(body, "issue"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	events := transport.Events()
	require.Len(t, events, 1)
	assert.Contains(t, capturedExceptionText(events[0]), "lookup failed for installation "+installationUUID)
	assert.Equal(t, installationUUID, events[0].Tags["installation_uuid"])

	var receipts []models.SentryWebhookReceipt
	require.NoError(t, database.Conn().Where("installation_uuid = ?", installationUUID).Find(&receipts).Error)
	require.Len(t, receipts, 1)
	assert.Equal(t, models.SentryWebhookOutcomeFailed, receipts[0].Outcome)
	assert.Equal(t, http.StatusInternalServerError, receipts[0].HTTPStatus)
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
		"", "", "", "test", "/app/templates", r.AuthService, false,
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
