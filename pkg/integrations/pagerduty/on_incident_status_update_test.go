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

func Test__OnIncidentStatusUpdate__HandleWebhook(t *testing.T) {
	trigger := &OnIncidentStatusUpdate{}

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
		body := []byte(`{"event":{"event_type":"incident.status_update_published","data":{"id":"su-1"}}}`)
		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1=invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
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
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("wrong event type -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"event_type":"incident.triggered","data":{"id":"inc-1"}}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1="+signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"event_type":"incident.status_update_published","agent":{"id":"agent-1"},"data":{"id":"su-1","message":"Status update message","incident":{"id":"incident-1","html_url":"https://example.com/incidents/incident-1"}}}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-PagerDuty-Signature", "v1="+signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "pagerduty.incident.status_update_published", payload.Type)
		assert.Equal(t, map[string]any{
			"agent": map[string]any{"id": "agent-1"},
			"status_update": map[string]any{
				"id":      "su-1",
				"message": "Status update message",
				"incident": map[string]any{
					"id":       "incident-1",
					"html_url": "https://example.com/incidents/incident-1",
				},
			},
			"incident": map[string]any{
				"id":       "incident-1",
				"html_url": "https://example.com/incidents/incident-1",
			},
		}, payload.Data)
	})
}

func Test__OnIncidentStatusUpdate__Setup(t *testing.T) {
	trigger := OnIncidentStatusUpdate{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("service is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnIncidentStatusUpdateConfiguration{},
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
			Configuration: OnIncidentStatusUpdateConfiguration{Service: "svc-1"},
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.WebhookRequests, 0)
	})
}
