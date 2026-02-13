package servicenow

import (
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
		"events": []string{"created"},
	}

	t.Run("missing X-Webhook-Secret -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing X-Webhook-Secret header")
	})

	t.Run("invalid secret -> 403", func(t *testing.T) {
		body := []byte(`{"event_type":"created","incident":{"sys_id":"abc123"}}`)
		headers := http.Header{}
		headers.Set("X-Webhook-Secret", "wrong-secret")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid webhook secret")
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte("invalid json")
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Webhook-Secret", secret)

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("missing event_type -> 400", func(t *testing.T) {
		body := []byte(`{"incident":{"sys_id":"abc123"}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Webhook-Secret", secret)

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing event_type")
	})

	t.Run("event type not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"event_type":"updated","incident":{"sys_id":"abc123"}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Webhook-Secret", secret)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid secret -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event_type":"created","incident":{"sys_id":"abc123","number":"INC0010001","short_description":"Test incident"}}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Webhook-Secret", secret)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "servicenow.incident.created", payload.Type)
		assert.Equal(t, map[string]any{
			"sys_id":            "abc123",
			"number":            "INC0010001",
			"short_description": "Test incident",
		}, payload.Data)
	})

	t.Run("body with control characters -> parsed successfully", func(t *testing.T) {
		body := []byte("{\"event_type\":\"created\",\"incident\":{\"sys_id\":\"abc123\",\"short_description\":\"test\\r\\nincident\"}}")
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Webhook-Secret", secret)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnIncident__HandleAction(t *testing.T) {
	trigger := OnIncident{}

	t.Run("resetAuthentication returns new secret", func(t *testing.T) {
		webhookCtx := &contexts.WebhookContext{Secret: "old-secret"}

		result, err := trigger.HandleAction(core.TriggerActionContext{
			Name:    "resetAuthentication",
			Webhook: webhookCtx,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, result["secret"])
	})

	t.Run("unknown action returns error", func(t *testing.T) {
		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name: "unknownAction",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "action unknownAction not supported")
	})
}

func Test__OnIncident__Setup(t *testing.T) {
	trigger := OnIncident{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("at least one event required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Webhook:       &contexts.WebhookContext{},
			Configuration: OnIncidentConfiguration{Events: []string{}},
		})

		require.ErrorContains(t, err, "at least one event type must be chosen")
	})

	t.Run("metadata already set -> no webhook setup is called", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: NodeMetadata{WebhookURL: "https://example.com/webhooks/123"},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Configuration: OnIncidentConfiguration{Events: []string{"created"}},
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "https://example.com/webhooks/123", metadata.WebhookURL)
	})

	t.Run("successful setup creates webhook", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Webhook:       &contexts.WebhookContext{},
			Configuration: OnIncidentConfiguration{Events: []string{"created"}},
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, metadata.WebhookURL)
	})
}
