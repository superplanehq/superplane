package posthog

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

var testLogger = logrus.NewEntry(logrus.New())

const validSecret = "test-webhook-secret"

func eventBody(name string) []byte {
	return []byte(`{"event":{"event":"` + name + `","uuid":"01912f3a","distinct_id":"user_1","properties":{"plan":"team"}},"person":{"id":"p1"},"project":{"id":"12345"}}`)
}

func tokenHeaders(token string) http.Header {
	headers := http.Header{}
	headers.Set(WebhookTokenHeader, token)
	return headers
}

func Test__OnEvent__Setup(t *testing.T) {
	trigger := &OnEvent{}

	t.Run("requests a webhook with the configured filters", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"projectId":          "12345",
				"events":             []string{"signup"},
				"filterTestAccounts": true,
			},
			Integration: integrationContext,
		})
		require.NoError(t, err)

		require.Len(t, integrationContext.WebhookRequests, 1)
		request, ok := integrationContext.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "12345", request.ProjectID)
		assert.Equal(t, []string{"signup"}, request.Events)
		assert.True(t, request.FilterTestAccounts)
	})

	t.Run("missing project returns error", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"events": []string{"signup"}},
			Integration:   integrationContext,
		})

		require.ErrorContains(t, err, "project is required")
		assert.Empty(t, integrationContext.WebhookRequests)
	})

	t.Run("invalid configuration format returns decode error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: "invalid-config",
			Integration:   &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__OnEvent__HandleWebhook(t *testing.T) {
	trigger := &OnEvent{}

	defaultConfig := map[string]any{"projectId": "12345"}

	t.Run("missing secret -> 403", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders(validSecret),
			Body:          eventBody("signup"),
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{},
			Events:        &contexts.EventContext{},
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "webhook secret is not available")
	})

	t.Run("missing token header -> 403", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          eventBody("signup"),
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        events,
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, WebhookTokenHeader)
		assert.Zero(t, events.Count())
	})

	t.Run("wrong token -> 403", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders("not-the-secret"),
			Body:          eventBody("signup"),
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        events,
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, WebhookTokenHeader)
		assert.Zero(t, events.Count())
	})

	t.Run("malformed body -> 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders(validSecret),
			Body:          []byte(`not json`),
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        &contexts.EventContext{},
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("payload without an event name -> 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders(validSecret),
			Body:          []byte(`{"person":{"id":"p1"}}`),
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        &contexts.EventContext{},
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing event name")
	})

	t.Run("valid delivery emits the payload", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders(validSecret),
			Body:          eventBody("signup"),
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        events,
			Logger:        testLogger,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())

		payload := events.Payloads[0]
		assert.Equal(t, "posthog.event", payload.Type)

		data, ok := payload.Data.(map[string]any)
		require.True(t, ok)
		event, ok := data["event"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "signup", event["event"])
		assert.NotNil(t, data["person"])
		assert.NotNil(t, data["project"])
	})

	t.Run("event outside the configured list is acknowledged without emitting", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders(validSecret),
			Body:          eventBody("pageview"),
			Configuration: map[string]any{"projectId": "12345", "events": []string{"signup"}},
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        events,
			Logger:        testLogger,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Zero(t, events.Count())
	})

	t.Run("event in the configured list is emitted", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders(validSecret),
			Body:          eventBody("signup"),
			Configuration: map[string]any{"projectId": "12345", "events": []string{"signup", "purchase"}},
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        events,
			Logger:        testLogger,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("no configured events accepts every event", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       tokenHeaders(validSecret),
			Body:          eventBody("some custom event"),
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{Secret: validSecret},
			Events:        events,
			Logger:        testLogger,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})
}

func Test__EventName(t *testing.T) {
	t.Run("reads the nested event name", func(t *testing.T) {
		payload := map[string]any{"event": map[string]any{"event": "signup"}}
		assert.Equal(t, "signup", eventName(payload))
	})

	t.Run("missing event object yields empty", func(t *testing.T) {
		assert.Empty(t, eventName(map[string]any{"person": map[string]any{}}))
	})

	t.Run("event that is not an object yields empty", func(t *testing.T) {
		assert.Empty(t, eventName(map[string]any{"event": "signup"}))
	})
}
