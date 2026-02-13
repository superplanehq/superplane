package grafana

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlertFiring__HandleWebhook(t *testing.T) {
	trigger := &OnAlertFiring{}

	payload := []byte(`{"status":"firing","alerts":[{"status":"firing"}]}`)

	t.Run("missing auth header when secret configured -> 401", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"sharedSecret": "secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusUnauthorized, code)
		assert.ErrorContains(t, err, "missing Authorization header")
	})

	t.Run("invalid auth header format -> 401", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Token secret")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       headers,
			Configuration: map[string]any{"sharedSecret": "secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusUnauthorized, code)
		assert.ErrorContains(t, err, "invalid Authorization header")
	})

	t.Run("invalid auth token -> 401", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer wrong")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       headers,
			Configuration: map[string]any{"sharedSecret": "secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusUnauthorized, code)
		assert.ErrorContains(t, err, "invalid Authorization token")
	})

	t.Run("valid auth token -> event emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer secret")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       headers,
			Configuration: map[string]any{"sharedSecret": "secret"},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		assert.Equal(t, "grafana.alert.firing", eventContext.Payloads[0].Type)
	})

	t.Run("no secret configured -> event emitted without auth header", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})
}
