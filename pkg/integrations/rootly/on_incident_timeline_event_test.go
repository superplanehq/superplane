package rootly

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncidentTimelineEvent__HandleWebhook(t *testing.T) {
	trigger := &OnIncidentTimelineEvent{}

	signatureFor := func(secret string, timestamp string, body []byte) string {
		payload := append([]byte(timestamp), body...)
		sig := computeHMACSHA256([]byte(secret), payload)
		return "t=" + timestamp + ",v1=" + sig
	}

	baseConfig := map[string]any{}

	t.Run("missing X-Rootly-Signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte("invalid json")
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("event type not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2026-02-10T15:34:36Z"},"data":{"id":"6bbb5dac-b5a4-4569-a398-0897f683cc54","event":"another try","kind":"event","visibility":"internal","occurred_at":"2026-02-10T15:34:29Z","created_at":"2026-02-10T15:34:29Z","incident_id":"inc-123"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      &contexts.MetadataContext{},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("incident_event.updated emits when metadata empty", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-123","issued_at":"2026-02-10T15:34:36Z"},"data":{"id":"ev-1","event":"Initial note","kind":"event","visibility":"internal","occurred_at":"2026-02-10T15:00:00Z","created_at":"2026-02-10T15:00:01Z","incident_id":"inc-123"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		metadata := &contexts.MetadataContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      metadata,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("incident_event.updated dedupes unchanged event", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-124","issued_at":"2026-02-10T15:40:00Z"},"data":{"id":"ev-1","event":"Initial note","kind":"event","visibility":"internal","occurred_at":"2026-02-10T15:00:00Z","created_at":"2026-02-10T15:00:01Z","updated_at":"2026-02-10T15:00:02Z","incident_id":"inc-123"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		metadata := &contexts.MetadataContext{Metadata: OnIncidentTimelineEventMetadata{EventStates: map[string]string{"ev-1": "2026-02-10T15:00:02Z"}}}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      metadata,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("visibility filter -> skips non-matching events", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-125","issued_at":"2026-02-10T15:40:00Z"},"data":{"id":"ev-2","event":"External update","kind":"event","visibility":"external","occurred_at":"2026-02-10T15:10:00Z","created_at":"2026-02-10T15:10:01Z","incident_id":"inc-123"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		metadata := &contexts.MetadataContext{Metadata: OnIncidentTimelineEventMetadata{EventStates: map[string]string{"ev-2": "2026-02-10T15:10:01Z"}}}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"visibility": "internal"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      metadata,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("incident_event.created -> emits even without baseline", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-200","issued_at":"2026-02-10T16:00:00Z"},"data":{"id":"ev-200","event":"Failover initiated","kind":"event","visibility":"internal","occurred_at":"2026-02-10T16:00:00Z","created_at":"2026-02-10T16:00:01Z","incident_id":"inc-200"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		metadata := &contexts.MetadataContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      metadata,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("incident_event.created flat payload -> emits", func(t *testing.T) {
		body := []byte(`{"event":{"id":"d390b7e6-2f80-4d71-92e3-5d19dc7f9c68","type":"incident_event.created","issued_at":"2026-02-17T05:54:29.581-08:00"},"data":{"id":"6bbb5dac-b5a4-4569-a398-0897f683cc54","event":"another try","event_raw":"another try","kind":"event","source":"web","visibility":"internal","occurred_at":"2026-02-17T05:54:29.522-08:00","created_at":"2026-02-17T05:54:29.522-08:00","updated_at":"2026-02-17T05:54:29.522-08:00","incident_id":"5dcbfe70-0416-469a-8629-be5353f4fd60"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      &contexts.MetadataContext{},
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "another try", payload["event"])
		assert.Equal(t, "another try", payload["event_raw"])
		assert.Equal(t, "d390b7e6-2f80-4d71-92e3-5d19dc7f9c68", payload["event_id"])
		assert.Equal(t, "incident_event.created", payload["event_type"])
		assert.Equal(t, "2026-02-17T05:54:29.581-08:00", payload["issued_at"])
		assert.Equal(t, "5dcbfe70-0416-469a-8629-be5353f4fd60", payload["incident_id"])
		assert.Equal(t, "2026-02-17T05:54:29.522-08:00", payload["updated_at"])
		incident := payload["incident"].(map[string]any)
		assert.Equal(t, "5dcbfe70-0416-469a-8629-be5353f4fd60", incident["id"])
	})

	t.Run("non-event kind -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-300","issued_at":"2026-02-10T16:10:00Z"},"data":{"id":"ev-300","event":"Note update","kind":"trail","visibility":"internal","occurred_at":"2026-02-10T16:10:00Z","created_at":"2026-02-10T16:10:01Z","incident_id":"inc-300"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      &contexts.MetadataContext{},
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("incident filters -> fetches incident and emits", func(t *testing.T) {
		now := time.Now().UTC().Format(time.RFC3339)
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-200","issued_at":"` + now + `"},"data":{"id":"ev-200","event":"Status update","kind":"event","source":"web","visibility":"internal","occurred_at":"` + now + `","created_at":"` + now + `","incident_id":"inc-200"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
  "data": {
    "id": "inc-200",
    "attributes": {
      "status": "resolved",
      "severity": "sev2"
    },
    "relationships": {
      "services": {"data": [{"type": "services", "id": "svc-1"}]},
      "groups": {"data": [{"type": "groups", "id": "team-1"}]}
    }
  },
  "included": [
    {"id": "svc-1", "type": "services", "attributes": {"name": "API"}},
    {"id": "team-1", "type": "groups", "attributes": {"name": "Platform"}}
  ]
}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"incidentStatus": []string{"resolved"}, "severity": []string{"sev2"}, "service": []string{"API"}, "team": []string{"Platform"}},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
			Metadata:      &contexts.MetadataContext{},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		require.Len(t, httpCtx.Requests, 1)

		payload := eventContext.Payloads[0].Data.(map[string]any)
		incident := payload["incident"].(map[string]any)
		assert.Equal(t, "inc-200", incident["id"])
		assert.Equal(t, "resolved", incident["status"])
		assert.Equal(t, "sev2", incident["severity"])
		assert.Equal(t, []string{"API"}, incident["services"])
		assert.Equal(t, []string{"Platform"}, incident["teams"])
	})
}

func Test__OnIncidentTimelineEvent__Setup(t *testing.T) {
	trigger := &OnIncidentTimelineEvent{}

	integrationCtx := &contexts.IntegrationContext{}
	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)

	webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	assert.Equal(t, []string{"incident_event.created", "incident_event.updated"}, webhookConfig.Events)
}
