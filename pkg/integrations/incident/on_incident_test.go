package incident

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type setupFirstWebhookContext struct {
	setupCalled bool
	secret      []byte
}

func (w *setupFirstWebhookContext) Setup() (string, error) {
	w.setupCalled = true
	return "https://example.com/api/v1/webhooks/webhook-123", nil
}

func (w *setupFirstWebhookContext) SetSecret(secret []byte) error {
	if !w.setupCalled {
		return fmt.Errorf("set secret called before setup")
	}
	w.secret = secret
	return nil
}

func (w *setupFirstWebhookContext) GetSecret() ([]byte, error) {
	return w.secret, nil
}

func (w *setupFirstWebhookContext) ResetSecret() ([]byte, []byte, error) {
	return w.secret, w.secret, nil
}

func (w *setupFirstWebhookContext) GetBaseURL() string {
	return "https://example.com"
}

func Test__OnIncident__HandleWebhook(t *testing.T) {
	trigger := &OnIncident{}

	validConfig := map[string]any{
		"events": []string{EventIncidentCreatedV2},
	}
	validSecret := "test-secret-32-bytes-long-enough!!"

	// Svix-style: signed = id + "." + timestamp + "." + body, sig = base64(HMAC-SHA256(signed, key))
	svixSignatureFor := func(secret, id, timestamp string, body []byte) string {
		signed := id + "." + timestamp + "." + string(body)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(signed))
		return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}

	t.Run("missing signing secret (no webhook secret) -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: map[string]any{"events": []string{EventIncidentCreatedV2}},
			Webhook:       &contexts.NodeWebhookContext{}, // no SetSecret called
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "signing secret is required")
	})

	t.Run("missing webhook headers -> 403", func(t *testing.T) {
		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: validConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.Error(t, err)
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"event_type":"public_incident.incident_created_v2","public_incident.incident_created_v2":{"id":"inc-1","name":"Test"}}`)
		headers := http.Header{}
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		headers.Set("webhook-id", "msg_123")
		headers.Set("webhook-timestamp", ts)
		headers.Set("webhook-signature", "v1,invalid")

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "signature")
	})

	t.Run("event type not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"event_type":"public_incident.incident_updated_v2","public_incident.incident_updated_v2":{"id":"inc-1","name":"Test"}}`)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		id := "msg_456"
		headers := http.Header{}
		headers.Set("webhook-id", id)
		headers.Set("webhook-timestamp", ts)
		headers.Set("webhook-signature", svixSignatureFor(validSecret, id, ts, body))

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"events": []string{EventIncidentCreatedV2}},
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature and event -> emit (secret from webhook)", func(t *testing.T) {
		body := []byte(`{"event_type":"public_incident.incident_created_v2","public_incident.incident_created_v2":{"id":"01FDAG4SAP5TYPT98WGR2N7W91","name":"Database sad","reference":"INC-123"}}`)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		id := "msg_789"
		headers := http.Header{}
		headers.Set("webhook-id", id)
		headers.Set("webhook-timestamp", ts)
		headers.Set("webhook-signature", svixSignatureFor(validSecret, id, ts, body))

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "incident.incident.created", eventContext.Payloads[0].Type)
		payload := eventContext.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "public_incident.incident_created_v2", payload["event_type"])
		incident, _ := payload["incident"].(map[string]any)
		require.NotNil(t, incident)
		assert.Equal(t, "01FDAG4SAP5TYPT98WGR2N7W91", incident["id"])
		assert.Equal(t, "Database sad", incident["name"])
	})

	t.Run("Svix envelope with data.incident -> emit", func(t *testing.T) {
		// Real incident.io payload shape: { "data": { "event_type": "...", "incident": {...} }, "type": "...", "timestamp": "..." }
		body := []byte(`{"type":"incident.incident.created","data":{"event_type":"public_incident.incident_created_v2","incident":{"id":"01FDAG4SAP5TYPT98WGR2N7W91","name":"Our database is sad","reference":"INC-123"}},"timestamp":"2026-01-19T12:00:00Z"}`)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		id := "msg_svix"
		headers := http.Header{}
		headers.Set("webhook-id", id)
		headers.Set("webhook-timestamp", ts)
		headers.Set("webhook-signature", svixSignatureFor(validSecret, id, ts, body))

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "incident.incident.created", eventContext.Payloads[0].Type)
		payload := eventContext.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "public_incident.incident_created_v2", payload["event_type"])
		incident, _ := payload["incident"].(map[string]any)
		require.NotNil(t, incident, "incident must be populated when payload uses data.incident")
		assert.Equal(t, "01FDAG4SAP5TYPT98WGR2N7W91", incident["id"])
		assert.Equal(t, "Our database is sad", incident["name"])
	})
}

func Test__OnIncident__HandleAction__SetSecret(t *testing.T) {
	trigger := &OnIncident{}

	t.Run("setSecret stores secret and updates metadata", func(t *testing.T) {
		webhookCtx := &setupFirstWebhookContext{}
		webhookCtx.setupCalled = true
		metadataCtx := &contexts.MetadataContext{}
		metadataCtx.Metadata = OnIncidentMetadata{WebhookURL: "https://example.com/webhook"}

		result, err := trigger.HandleAction(core.TriggerActionContext{
			Name:       "setSecret",
			Parameters: map[string]any{"webhookSigningSecret": "whsec_abc123"},
			Webhook:    webhookCtx,
			Metadata:   metadataCtx,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, true, result["ok"])
		assert.Equal(t, true, result["signingSecretConfigured"])
		assert.Equal(t, "whsec_abc123", string(webhookCtx.secret))

		metadata, ok := metadataCtx.Metadata.(OnIncidentMetadata)
		require.True(t, ok)
		assert.True(t, metadata.SigningSecretConfigured)
		assert.Equal(t, "https://example.com/webhook", metadata.WebhookURL)
	})

	t.Run("setSecret with empty string clears secret", func(t *testing.T) {
		webhookCtx := &setupFirstWebhookContext{}
		webhookCtx.setupCalled = true
		webhookCtx.secret = []byte("existing")
		metadataCtx := &contexts.MetadataContext{}
		metadataCtx.Metadata = OnIncidentMetadata{WebhookURL: "https://example.com/webhook", SigningSecretConfigured: true}

		result, err := trigger.HandleAction(core.TriggerActionContext{
			Name:       "setSecret",
			Parameters: map[string]any{"webhookSigningSecret": ""},
			Webhook:    webhookCtx,
			Metadata:   metadataCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, true, result["ok"])
		assert.Equal(t, false, result["signingSecretConfigured"])
		assert.Equal(t, []byte{}, webhookCtx.secret)

		metadata, ok := metadataCtx.Metadata.(OnIncidentMetadata)
		require.True(t, ok)
		assert.False(t, metadata.SigningSecretConfigured)
	})

	t.Run("setSecret without webhook returns error", func(t *testing.T) {
		result, err := trigger.HandleAction(core.TriggerActionContext{
			Name:       "setSecret",
			Parameters: map[string]any{"webhookSigningSecret": "whsec_xyz"},
			Webhook:    nil,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "webhook is not available")
	})

	t.Run("unsupported action returns error", func(t *testing.T) {
		result, err := trigger.HandleAction(core.TriggerActionContext{
			Name:    "unknownAction",
			Webhook: &setupFirstWebhookContext{setupCalled: true},
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "not supported")
	})
}

func Test__OnIncident__Setup(t *testing.T) {
	trigger := &OnIncident{}

	t.Run("at least one event required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnIncidentConfiguration{Events: []string{}},
		})

		require.ErrorContains(t, err, "at least one event type must be chosen")
	})

	t.Run("valid config requests webhook", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnIncidentConfiguration{Events: []string{EventIncidentCreatedV2}},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		req := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, []string{EventIncidentCreatedV2}, req.Events)
	})

	t.Run("setup webhook URL and metadata without secret", func(t *testing.T) {
		webhookCtx := &setupFirstWebhookContext{}
		metadataCtx := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Webhook:       webhookCtx,
			Metadata:      metadataCtx,
			Configuration: OnIncidentConfiguration{Events: []string{EventIncidentCreatedV2}},
		})

		require.NoError(t, err)
		assert.True(t, webhookCtx.setupCalled)
		// Secret is set only via setSecret action, not in Setup
		assert.Nil(t, webhookCtx.secret)

		metadata, ok := metadataCtx.Metadata.(OnIncidentMetadata)
		require.True(t, ok)
		assert.Equal(t, "https://example.com/api/v1/webhooks/webhook-123", metadata.WebhookURL)
		assert.False(t, metadata.SigningSecretConfigured)
	})
}
