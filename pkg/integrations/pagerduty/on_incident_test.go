package pagerduty

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncident__HandleWebhook(t *testing.T) {
	trigger := &OnIncident{}

	validConfig := map[string]any{
		"events":    []string{"incident.triggered"},
		"urgencies": []string{"high", "low"},
	}

	signatureFor := func(secret string, body []byte) string {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		return fmt.Sprintf("%x", h.Sum(nil))
	}

	t.Run("missing X-PagerDuty-Signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing signature")
	})

	t.Run("invalid signature format -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Events:  &contexts.EventContext{},
			Webhook: &contexts.NodeWebhookContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"event":{"event_type":"incident.triggered","data":{"urgency":"high"}}}`)
		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1=invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte("invalid json")
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1="+signatureFor(secret, body))

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("event type not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"event_type":"incident.acknowledged","data":{"urgency":"high"}}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1="+signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("urgency not allowed -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"event_type":"incident.triggered","data":{"urgency":"low"}}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1="+signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"events": []string{"incident.triggered"}, "urgencies": []string{"high"}},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"event_type":"incident.triggered","agent":{"id":"agent-1"},"data":{"id":"incident-1","urgency":"high"}}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1="+signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "pagerduty.incident.triggered", payload.Type)
		assert.Equal(t, map[string]any{
			"agent": map[string]any{"id": "agent-1"},
			"incident": map[string]any{
				"id":      "incident-1",
				"urgency": "high",
			},
		}, payload.Data)
	})
}

func Test__OnIncident__Setup(t *testing.T) {
	trigger := OnIncident{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("at least one event required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnIncidentConfiguration{Events: []string{}, Service: "svc-1"},
		})

		require.ErrorContains(t, err, "at least one event type must be chosen")
	})

	t.Run("service is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnIncidentConfiguration{Events: []string{"incident.triggered"}},
		})

		require.ErrorContains(t, err, "service is required")
	})

	t.Run("metadata already set -> no webhook request is made", func(t *testing.T) {
		service := &Service{ID: "svc-1", Name: "test-service"}
		integrationCtx := &contexts.IntegrationContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: NodeMetadata{Service: service},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadataCtx,
			Configuration: OnIncidentConfiguration{Events: []string{"incident.triggered"}, Service: "svc-1"},
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.WebhookRequests, 0)
	})
}
