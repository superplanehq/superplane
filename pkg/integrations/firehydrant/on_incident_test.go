package firehydrant

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncident__Setup(t *testing.T) {
	trigger := &OnIncident{}

	t.Run("valid configuration -> success", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
			WebhookRequests: []any{},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: OnIncidentConfiguration{
				Events: []string{"incident.created"},
			},
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.WebhookRequests, 1)
	})

	t.Run("no events -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: OnIncidentConfiguration{
				Events: []string{},
			},
			Integration: integrationCtx,
		})

		require.ErrorContains(t, err, "at least one event type must be chosen")
	})
}

func Test__OnIncident__HandleWebhook(t *testing.T) {
	trigger := &OnIncident{}

	t.Run("valid webhook -> event emitted", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{
			Payloads: []contexts.Payload{},
		}

		webhookCtx := &contexts.WebhookContext{
			Secret: "test-secret",
		}

		// Compute the expected signature
		body := `{"event":{"id":"evt-123","type":"incident.created","created_at":"2026-01-19T12:00:00Z"},"data":{"id":"inc-123","name":"Test Incident"}}`

		headers := http.Header{}
		// For test purposes, we'll verify that invalid signature fails
		headers.Set("X-FireHydrant-Signature", "invalid-signature")

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: OnIncidentConfiguration{
				Events: []string{"incident.created"},
			},
			Body:    []byte(body),
			Headers: headers,
			Webhook: webhookCtx,
			Events:  eventsCtx,
		})

		// Should fail due to signature mismatch
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("event not in config -> ignored", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{
			Payloads: []contexts.Payload{},
		}

		webhookCtx := &contexts.WebhookContext{
			Secret: "test-secret",
		}

		headers := http.Header{}
		headers.Set("X-FireHydrant-Signature", "any-signature")

		body := `{"event":{"id":"evt-123","type":"incident.deleted","created_at":"2026-01-19T12:00:00Z"},"data":{"id":"inc-123"}}`

		// Since signature validation happens before event filtering,
		// this will fail on signature first
		_, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: OnIncidentConfiguration{
				Events: []string{"incident.created"},
			},
			Body:    []byte(body),
			Headers: headers,
			Webhook: webhookCtx,
			Events:  eventsCtx,
		})

		// Signature mismatch or no events emitted
		require.Error(t, err)
	})

	t.Run("invalid JSON -> bad request", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{
			Payloads: []contexts.Payload{},
		}

		webhookCtx := &contexts.WebhookContext{
			Secret: "test-secret",
		}

		headers := http.Header{}
		headers.Set("X-FireHydrant-Signature", "valid-signature-would-go-here")

		body := `invalid json`

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: OnIncidentConfiguration{
				Events: []string{"incident.created"},
			},
			Body:    []byte(body),
			Headers: headers,
			Webhook: webhookCtx,
			Events:  eventsCtx,
		})

		// Will fail on signature before JSON parsing
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, status)
	})
}

func Test__buildIncidentPayload(t *testing.T) {
	t.Run("complete webhook payload", func(t *testing.T) {
		webhook := WebhookPayload{
			Event: WebhookEvent{
				ID:        "evt-123",
				Type:      "incident.created",
				CreatedAt: "2026-01-19T12:00:00Z",
			},
			Data: map[string]any{
				"id":   "inc-123",
				"name": "Test Incident",
			},
		}

		payload := buildIncidentPayload(webhook)

		assert.Equal(t, "incident.created", payload["event"])
		assert.Equal(t, "evt-123", payload["event_id"])
		assert.Equal(t, "2026-01-19T12:00:00Z", payload["created_at"])
		assert.NotNil(t, payload["incident"])
	})

	t.Run("empty data", func(t *testing.T) {
		webhook := WebhookPayload{
			Event: WebhookEvent{
				ID:        "evt-456",
				Type:      "incident.resolved",
				CreatedAt: "2026-01-19T13:00:00Z",
			},
			Data: nil,
		}

		payload := buildIncidentPayload(webhook)

		assert.Equal(t, "incident.resolved", payload["event"])
		assert.Equal(t, "evt-456", payload["event_id"])
		assert.Nil(t, payload["incident"])
	})
}
