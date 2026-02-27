package honeycomb

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlertFired__Setup(t *testing.T) {
	trigger := OnAlertFired{}

	t.Run("missing datasetSlug -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"trigger": "High Error Rate",
			},
		})
		require.ErrorContains(t, err, "datasetSlug is required")
	})

	t.Run("missing trigger -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"datasetSlug": "production",
			},
		})
		require.ErrorContains(t, err, "trigger is required")
	})

	t.Run("no integration -> returns nil without requesting webhook", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration: nil,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"datasetSlug": "production",
				"trigger":     "High Error Rate",
			},
		})
		assert.NoError(t, err)
	})
}

func Test__OnAlertFired__HandleWebhook(t *testing.T) {
	trigger := &OnAlertFired{}

	validConfig := map[string]any{
		"datasetSlug": "production",
		"trigger":     "High Error Rate",
	}

	body := []byte(`{"id":"trigger-abc","name":"High Error Rate","status":"TRIGGERED"}`)

	t.Run("missing token -> 401", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
			Metadata:      &contexts.MetadataContext{},
		})
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.ErrorContains(t, err, "missing webhook token")
	})

	t.Run("invalid token -> 403", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "wrong-secret-xx")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
			Metadata:      &contexts.MetadataContext{},
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid webhook token")
	})

	t.Run("invalid JSON body -> falls back to raw payload and emits", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "test-secret")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          []byte(`not valid json`),
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
			Metadata:      &contexts.MetadataContext{},
		})
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("valid token, triggerID matches -> emits", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "test-secret")

		meta := &contexts.MetadataContext{}
		_ = meta.Set(OnAlertFiredNodeMetadata{TriggerID: "trigger-abc"})

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
			Metadata:      meta,
		})
		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
		assert.Equal(t, "honeycomb.alert.fired", events.Payloads[0].Type)
	})

	t.Run("valid token, triggerID does not match -> no emit", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "test-secret")

		meta := &contexts.MetadataContext{}
		_ = meta.Set(OnAlertFiredNodeMetadata{TriggerID: "different-trigger"})

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
			Metadata:      meta,
		})
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("valid token, no metadata -> emits without filter", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "test-secret")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
			Metadata:      &contexts.MetadataContext{},
		})
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("bearer authorization header -> emits", func(t *testing.T) {
		h := http.Header{}
		h.Set("Authorization", "Bearer test-secret")

		meta := &contexts.MetadataContext{}
		_ = meta.Set(OnAlertFiredNodeMetadata{TriggerID: "trigger-abc"})

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
			Metadata:      meta,
		})
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}
