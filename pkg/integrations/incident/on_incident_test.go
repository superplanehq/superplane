package incident

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncident__HandleWebhook(t *testing.T) {
	trigger := &OnIncident{}

	validConfig := map[string]any{
		"events":        []string{EventIncidentCreatedV2},
		"signingSecret": "test-secret-32-bytes-long-enough!!",
	}

	// Svix-style: signed = id + "." + timestamp + "." + body, sig = base64(HMAC-SHA256(signed, key))
	svixSignatureFor := func(secret, id, timestamp string, body []byte) string {
		signed := id + "." + timestamp + "." + string(body)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(signed))
		return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}

	t.Run("missing signing secret in config -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: map[string]any{"events": []string{EventIncidentCreatedV2}}, // no signingSecret
			Webhook:       &contexts.WebhookContext{},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "signing secret is required")
	})

	t.Run("missing webhook headers -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{},
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

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "signature")
	})

	t.Run("event type not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"event_type":"public_incident.incident_updated_v2","public_incident.incident_updated_v2":{"id":"inc-1","name":"Test"}}`)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		id := "msg_456"
		secret := "test-secret-32-bytes-long-enough!!"
		headers := http.Header{}
		headers.Set("webhook-id", id)
		headers.Set("webhook-timestamp", ts)
		headers.Set("webhook-signature", svixSignatureFor(secret, id, ts, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"events": []string{EventIncidentCreatedV2}, "signingSecret": secret},
			Webhook:       &contexts.WebhookContext{},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature and event -> emit", func(t *testing.T) {
		body := []byte(`{"event_type":"public_incident.incident_created_v2","public_incident.incident_created_v2":{"id":"01FDAG4SAP5TYPT98WGR2N7W91","name":"Database sad","reference":"INC-123"}}`)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		id := "msg_789"
		secret := "test-secret-32-bytes-long-enough!!"
		headers := http.Header{}
		headers.Set("webhook-id", id)
		headers.Set("webhook-timestamp", ts)
		headers.Set("webhook-signature", svixSignatureFor(secret, id, ts, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{},
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
}
