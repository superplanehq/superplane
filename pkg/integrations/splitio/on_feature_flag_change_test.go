package splitio

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

func Test__OnFeatureFlagChange__HandleWebhook(t *testing.T) {
	trigger := &OnFeatureFlagChange{}

	defaultConfig := map[string]any{}

	t.Run("valid split event -> emit", func(t *testing.T) {
		body := []byte(`{"name":"my-feature","type":"split","environmentName":"production","description":"User updated targeting rules","editor":"user@example.com","time":1709308800000}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "splitio.flag.updated", eventContext.Payloads[0].Type)
		payload, ok := eventContext.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-feature", payload["name"])
		assert.Equal(t, "production", payload["environmentName"])
	})

	t.Run("non-split type -> no emit", func(t *testing.T) {
		body := []byte(`{"name":"some-segment","type":"segment","environmentName":"production"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("empty type is accepted -> emit", func(t *testing.T) {
		body := []byte(`{"name":"my-flag","environmentName":"staging","description":"Flag killed"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "splitio.flag.killed", eventContext.Payloads[0].Type)
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`not-json`),
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("flag matches predicate -> emit", func(t *testing.T) {
		body := []byte(`{"name":"my-feature","type":"split","environmentName":"production","description":"Flag created"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"flags": []map[string]any{{"type": "equals", "value": "my-feature"}},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("flag does not match predicate -> no emit", func(t *testing.T) {
		body := []byte(`{"name":"other-flag","type":"split","environmentName":"production","description":"Flag updated"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"flags": []map[string]any{{"type": "equals", "value": "my-feature"}},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("flag matches pattern predicate -> emit", func(t *testing.T) {
		body := []byte(`{"name":"beta-feature-123","type":"split","environmentName":"production","description":"Flag updated"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"flags": []map[string]any{{"type": "matches", "value": "^beta-.*"}},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("environment matches predicate -> emit", func(t *testing.T) {
		body := []byte(`{"name":"my-feature","type":"split","environmentName":"production","description":"Flag updated"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"environments": []map[string]any{{"type": "equals", "value": "production"}},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("environment does not match predicate -> no emit", func(t *testing.T) {
		body := []byte(`{"name":"my-feature","type":"split","environmentName":"development","description":"Flag updated"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"environments": []map[string]any{{"type": "equals", "value": "production"}},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("no filters -> accept all events", func(t *testing.T) {
		body := []byte(`{"name":"any-flag","type":"split","environmentName":"staging","description":"Flag restored"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "splitio.flag.restored", eventContext.Payloads[0].Type)
	})

	t.Run("infers killed action from description", func(t *testing.T) {
		body := []byte(`{"name":"my-flag","type":"split","environmentName":"production","description":"User killed the flag"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "splitio.flag.killed", eventContext.Payloads[0].Type)
	})

	t.Run("infers created action from description", func(t *testing.T) {
		body := []byte(`{"name":"new-flag","type":"split","environmentName":"production","description":"Flag created by user"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "splitio.flag.created", eventContext.Payloads[0].Type)
	})

	t.Run("no description -> generic changed type", func(t *testing.T) {
		body := []byte(`{"name":"my-flag","type":"split","environmentName":"production"}`)

		wc := &contexts.NodeWebhookContext{}
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "splitio.flag", eventContext.Payloads[0].Type)
	})
}

func Test__OnFeatureFlagChange__Setup(t *testing.T) {
	trigger := &OnFeatureFlagChange{}

	t.Run("sets up webhook and stores URL in metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		webhookCtx := &contexts.NodeWebhookContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Webhook:       webhookCtx,
			Configuration: OnFeatureFlagChangeConfiguration{},
		})

		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(OnFeatureFlagChangeMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, metadata.URL)
	})

	t.Run("skips setup when URL already exists", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: OnFeatureFlagChangeMetadata{URL: "https://existing-url.com"},
		}
		webhookCtx := &contexts.NodeWebhookContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Webhook:       webhookCtx,
			Configuration: OnFeatureFlagChangeConfiguration{},
		})

		require.NoError(t, err)
	})
}
