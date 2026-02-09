package sentry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssueEvent__Setup(t *testing.T) {
	trigger := &OnIssueEvent{}
	integration := &contexts.IntegrationContext{}

	t.Run("missing events -> error", func(t *testing.T) {
		ctx := core.TriggerContext{
			Configuration: map[string]any{},
			Integration:   integration,
		}
		err := trigger.Setup(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one event")
	})

	t.Run("valid config -> requests webhook", func(t *testing.T) {
		ctx := core.TriggerContext{
			Configuration: map[string]any{
				"events": []string{"created", "resolved"},
			},
			Integration: integration,
		}
		err := trigger.Setup(ctx)
		assert.NoError(t, err)
		assert.Len(t, integration.WebhookRequests, 1)
		cfg, ok := integration.WebhookRequests[0].(WebhookConfiguration)
		assert.True(t, ok)
		assert.Equal(t, []string{"created", "resolved"}, cfg.Events)
	})
}

func Test__OnIssueEvent__HandleWebhook(t *testing.T) {
	trigger := &OnIssueEvent{}

	config := map[string]any{
		"events": []string{"created", "resolved"},
	}

	t.Run("missing Sentry-Hook-Resource or wrong resource -> 200 no emit", func(t *testing.T) {
		body := []byte(`{"action":"created","data":{"issue":{"id":"1"}}}`)
		headers := http.Header{}
		headers.Set("Sentry-Hook-Resource", "metric_alert")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("invalid JSON -> 400", func(t *testing.T) {
		body := []byte("invalid")
		headers := http.Header{}
		headers.Set("Sentry-Hook-Resource", "issue")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "invalid JSON")
	})

	t.Run("action not in config -> 200 no emit", func(t *testing.T) {
		body := []byte(`{"action":"assigned","data":{"issue":{"id":"1"}},"installation":{},"actor":{}}`)
		headers := http.Header{}
		headers.Set("Sentry-Hook-Resource", "issue")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature and action created -> emit", func(t *testing.T) {
		body := []byte(`{"action":"created","data":{"issue":{"id":"123","title":"Error"}},"installation":{"uuid":"inst-1"},"actor":{"name":"Sentry"}}`)
		headers := http.Header{}
		headers.Set("Sentry-Hook-Resource", "issue")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "sentry.issue.created", eventContext.Payloads[0].Type)
		data, ok := eventContext.Payloads[0].Data.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "created", data["action"])
		issue, ok := data["issue"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "123", issue["id"])
	})
}
