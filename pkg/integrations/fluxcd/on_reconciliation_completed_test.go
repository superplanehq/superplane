package fluxcd

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type webhookSecretContext struct {
	secret []byte
	err    error
}

func (w *webhookSecretContext) Setup() (string, error) {
	return "", nil
}

func (w *webhookSecretContext) SetSecret(secret []byte) error {
	w.secret = secret
	return nil
}

func (w *webhookSecretContext) GetSecret() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}

	return w.secret, nil
}

func (w *webhookSecretContext) ResetSecret() ([]byte, []byte, error) {
	return nil, nil, nil
}

func (w *webhookSecretContext) GetBaseURL() string {
	return "http://localhost:3000/api/v1"
}

func Test__OnReconciliationCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnReconciliationCompleted{}

	successPayload := []byte(`{
		"involvedObject": {
			"kind": "Kustomization",
			"namespace": "flux-system",
			"name": "my-app"
		},
		"severity": "info",
		"reason": "ReconciliationSucceeded",
		"message": "Reconciliation finished",
		"timestamp": "2026-01-15T10:30:00Z"
	}`)

	t.Run("missing auth header when secret configured -> 401", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          successPayload,
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
			Body:          successPayload,
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
			Body:          successPayload,
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
			Body:          successPayload,
			Headers:       headers,
			Configuration: map[string]any{"sharedSecret": "secret"},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "fluxcd.reconciliation", eventContext.Payloads[0].Type)
	})

	t.Run("no secret configured -> event emitted without auth header", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          successPayload,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("error severity -> no event emitted", func(t *testing.T) {
		errorPayload := []byte(`{
			"involvedObject": {"kind": "Kustomization", "namespace": "flux-system", "name": "my-app"},
			"severity": "error",
			"reason": "ReconciliationFailed",
			"message": "Reconciliation failed"
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          errorPayload,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("non-succeeded reason -> no event emitted", func(t *testing.T) {
		progressPayload := []byte(`{
			"involvedObject": {"kind": "Kustomization", "namespace": "flux-system", "name": "my-app"},
			"severity": "info",
			"reason": "Progressing",
			"message": "Reconciliation in progress"
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          progressPayload,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("kind filter matches -> event emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    successPayload,
			Headers: http.Header{},
			Configuration: map[string]any{
				"kinds": []any{"Kustomization"},
			},
			Events: eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("kind filter does not match -> no event emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    successPayload,
			Headers: http.Header{},
			Configuration: map[string]any{
				"kinds": []any{"HelmRelease"},
			},
			Events: eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("empty body -> 400", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte{},
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "empty body")
	})

	t.Run("invalid JSON -> 400", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`not json`),
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("webhook secret retrieval error returns 500", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          successPayload,
			Headers:       http.Header{},
			Configuration: map[string]any{"sharedSecret": "secret"},
			Webhook: &webhookSecretContext{
				err: errors.New("storage unavailable"),
			},
			Events: eventContext,
		})

		require.Equal(t, http.StatusInternalServerError, code)
		require.ErrorContains(t, err, "error getting webhook secret")
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("falls back to configuration secret when webhook secret is empty", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer secret")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          successPayload,
			Headers:       headers,
			Configuration: map[string]any{"sharedSecret": "secret"},
			Webhook: &webhookSecretContext{
				secret: []byte(""),
			},
			Events: eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "fluxcd.reconciliation", eventContext.Payloads[0].Type)
	})

	t.Run("HelmRelease reconciliation succeeded -> event emitted", func(t *testing.T) {
		helmPayload := []byte(`{
			"involvedObject": {
				"kind": "HelmRelease",
				"namespace": "default",
				"name": "nginx"
			},
			"severity": "info",
			"reason": "ReconciliationSucceeded",
			"message": "Release reconciliation succeeded"
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          helmPayload,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnReconciliationCompleted__Setup(t *testing.T) {
	trigger := &OnReconciliationCompleted{}

	t.Run("requests webhook through integration and stores webhook url metadata", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{}
		metadataContext := &contexts.MetadataContext{}
		webhookContext := &contexts.NodeWebhookContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"sharedSecret": "secret"},
			Integration:   integrationContext,
			Metadata:      metadataContext,
			Webhook:       webhookContext,
		})

		require.NoError(t, err)
		require.Len(t, integrationContext.WebhookRequests, 1)

		requestConfig, ok := integrationContext.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		require.Equal(t, "secret", requestConfig.SharedSecret)
		require.NotEmpty(t, requestConfig.WebhookBindingKey)

		metadata, ok := metadataContext.Metadata.(map[string]any)
		require.True(t, ok)
		require.NotEmpty(t, metadata["webhookUrl"])
		require.Equal(t, requestConfig.WebhookBindingKey, metadata["webhookBindingKey"])
	})

	t.Run("reuses existing webhook binding key", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{}
		metadataContext := &contexts.MetadataContext{
			Metadata: map[string]any{
				"webhookBindingKey": "node-1-key",
			},
		}
		webhookContext := &contexts.NodeWebhookContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"sharedSecret": "secret"},
			Integration:   integrationContext,
			Metadata:      metadataContext,
			Webhook:       webhookContext,
		})

		require.NoError(t, err)
		require.Len(t, integrationContext.WebhookRequests, 1)

		requestConfig, ok := integrationContext.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		require.Equal(t, "node-1-key", requestConfig.WebhookBindingKey)
	})

	t.Run("missing integration context returns error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"sharedSecret": "secret"},
		})

		require.ErrorContains(t, err, "missing integration context")
	})

	t.Run("missing webhook context returns error", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"sharedSecret": "secret"},
			Integration:   integrationContext,
		})

		require.ErrorContains(t, err, "missing webhook context")
		require.Len(t, integrationContext.WebhookRequests, 0)
	})
}
