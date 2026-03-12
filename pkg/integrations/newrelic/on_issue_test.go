package newrelic

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("no statuses -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"statuses": []string{}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{},
		})

		require.ErrorContains(t, err, "at least one status must be selected")
	})

	t.Run("valid setup requests shared webhook", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}},
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}
	payload := []byte(`{
		"issueId": "MXxBSXxJU1NVRXwxMjM0NTY3ODk",
		"issueUrl": "https://one.newrelic.com/alerts-ai/issues/MXxBSXxJU1NVRXwxMjM0NTY3ODk",
		"title": "High CPU usage on production server",
		"priority": "CRITICAL",
		"state": "ACTIVATED",
		"policyName": "Production Infrastructure",
		"conditionName": "CPU usage > 90%",
		"accountId": 1234567,
		"createdAt": 1704067200000,
		"updatedAt": 1704067260000,
		"sources": ["newrelic"]
	}`)

	t.Run("valid payload -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}},
			Webhook:       &contexts.NodeWebhookContext{},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
		assert.Equal(t, NewRelicIssuePayloadType, eventsCtx.Payloads[0].Type)
		data := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "MXxBSXxJU1NVRXwxMjM0NTY3ODk", data["issueId"])
		assert.Equal(t, "ACTIVATED", data["state"])
		assert.Equal(t, "CRITICAL", data["priority"])
	})

	t.Run("filtered by status -> skipped", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{"CLOSED"}},
			Webhook:       &contexts.NodeWebhookContext{},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("filtered by priority -> skipped", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}, "priorities": []string{"LOW"}},
			Webhook:       &contexts.NodeWebhookContext{},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("malformed JSON -> 400 error", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not-json"),
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}},
			Webhook:       &contexts.NodeWebhookContext{},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "failed to parse request body")
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("matching priority -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}, "priorities": []string{"CRITICAL", "HIGH"}},
			Webhook:       &contexts.NodeWebhookContext{},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("missing bearer token when secret configured -> 403", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}},
			Webhook:       &contexts.NodeWebhookContext{Secret: "my-secret"},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "missing bearer authorization")
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("invalid bearer token -> 403", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		headers := http.Header{}
		headers.Set("Authorization", "Bearer wrong-token")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       headers,
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}},
			Webhook:       &contexts.NodeWebhookContext{Secret: "my-secret"},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid bearer token")
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("empty secret -> no auth required", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}},
			Webhook:       &contexts.NodeWebhookContext{},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("valid bearer token -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		headers := http.Header{}
		headers.Set("Authorization", "Bearer my-secret")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       headers,
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}},
			Webhook:       &contexts.NodeWebhookContext{Secret: "my-secret"},
			Integration:   &contexts.IntegrationContext{},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
	})
}

func Test__OnIssue__Validation(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("invalid priority -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"statuses": []string{"ACTIVATED"}, "priorities": []string{"BANANA"}},
			Integration:   &contexts.IntegrationContext{},
			Webhook:       &contexts.NodeWebhookContext{},
		})

		require.ErrorContains(t, err, "invalid priority")
	})
}
