package honeycomb

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlertFired__Setup(t *testing.T) {
	trigger := OnAlertFired{}

	t.Run("requests shared webhook with empty config", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"alertName": "High Error Rate",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		req, ok := integrationCtx.WebhookRequests[0].(map[string]any)
		require.True(t, ok)
		require.Len(t, req, 0) // empty config => shared webhook
	})
}

func Test__OnAlertFired__HandleWebhook(t *testing.T) {
	trigger := &OnAlertFired{}

	validConfig := map[string]any{
		"alertName": "High Error Rate",
	}

	body := []byte(`{"alert":{"name":"High Error Rate"},"status":"firing"}`)

	t.Run("missing token header -> 401", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		require.Equal(t, http.StatusUnauthorized, code)
		require.ErrorContains(t, err, "missing webhook token")
	})

	t.Run("invalid token -> 403", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "wrong-secret")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		require.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid webhook token")
	})

	t.Run("valid token but alertName does not match -> no emit", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "test-secret")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          []byte(`{"alert":{"name":"Different"}}`),
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 0, events.Count())
	})

	t.Run("valid token + match -> emits", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "test-secret")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		require.Equal(t, "honeycomb.alert.fired", events.Payloads[0].Type)
	})

	t.Run("authorization bearer token works -> emits", func(t *testing.T) {
		h := http.Header{}
		h.Set("Authorization", "Bearer test-secret")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        events,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})

	t.Run("custom token header starting with bearer is not stripped -> emits", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Honeycomb-Webhook-Token", "bearer test-secret")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "bearer test-secret"},
			Events:        events,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})

	t.Run("shared secret header starting with bearer is not stripped -> emits", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Shared-Secret", "Bearer test-secret")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       h,
			Body:          body,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "Bearer test-secret"},
			Events:        events,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})

}
