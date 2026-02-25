package launchdarkly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type testWebhookContext struct {
	setupCalled bool
	secret      []byte
}

func (w *testWebhookContext) Setup() (string, error) {
	w.setupCalled = true
	return "https://example.com/api/v1/webhooks/webhook-123", nil
}

func (w *testWebhookContext) SetSecret(secret []byte) error {
	if !w.setupCalled {
		return fmt.Errorf("set secret called before setup")
	}
	w.secret = secret
	return nil
}

func (w *testWebhookContext) GetSecret() ([]byte, error) {
	return w.secret, nil
}

func (w *testWebhookContext) ResetSecret() ([]byte, []byte, error) {
	return w.secret, w.secret, nil
}

func (w *testWebhookContext) GetBaseURL() string {
	return "https://example.com"
}

// hmacSignature computes the LaunchDarkly HMAC-SHA256 hex signature for a body.
func hmacSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func Test__OnFeatureFlagChange__HandleWebhook(t *testing.T) {
	trigger := &OnFeatureFlagChange{}

	validConfig := map[string]any{
		"events": []string{KindFlag},
	}
	validSecret := "test-signing-secret"

	t.Run("missing signing secret -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "signing secret is required")
	})

	t.Run("missing X-LD-Signature header -> 403", func(t *testing.T) {
		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          []byte(`{}`),
			Configuration: validConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing X-LD-Signature header")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"Test Flag"}`)
		headers := http.Header{}
		headers.Set("X-LD-Signature", "invalidsignature")

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("event kind not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"kind":"project","name":"Some Project"}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"events": []string{KindFlag}},
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature and flag event -> emit", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","titleVerb":"turned on the flag","title":"User turned on the flag My Feature","accesses":[{"action":"updateOn","resource":"proj/default:env/test:flag/my-feature"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag.updateOn", eventContext.Payloads[0].Type)
		payload, ok := eventContext.Payloads[0].Data.(map[string]any)
		require.True(t, ok, "expected Payloads[0].Data to be map[string]any")
		assert.Equal(t, "flag", payload["kind"])
		assert.Equal(t, "My Feature", payload["name"])
	})

	t.Run("flag event without accesses -> emit with kind-only type", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"Simple Flag"}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag", eventContext.Payloads[0].Type)
	})

	t.Run("action not in configured actions -> no emit", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","accesses":[{"action":"updateRules","resource":"proj/default:env/test:flag/my-feature"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"events": []string{KindFlag}, "actions": []string{ActionUpdateOn}},
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("action in configured actions -> emit", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","accesses":[{"action":"updateOn","resource":"proj/default:env/test:flag/my-feature"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"events": []string{KindFlag}, "actions": []string{ActionUpdateOn}},
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag.updateOn", eventContext.Payloads[0].Type)
	})

	t.Run("empty actions config -> all actions accepted", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","accesses":[{"action":"deleteFlag","resource":"proj/default:flag/my-feature"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"events": []string{KindFlag}},
			Webhook:       wc,
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag.deleteFlag", eventContext.Payloads[0].Type)
	})

	t.Run("missing kind in payload -> 400", func(t *testing.T) {
		body := []byte(`{"name":"No Kind Field"}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.WebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing kind in payload")
	})
}

func Test__OnFeatureFlagChange__HandleAction(t *testing.T) {
	trigger := &OnFeatureFlagChange{}

	t.Run("unsupported action returns error", func(t *testing.T) {
		result, err := trigger.HandleAction(core.TriggerActionContext{
			Name:    "setSecret",
			Webhook: &testWebhookContext{setupCalled: true},
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "not supported")
	})
}

func Test__OnFeatureFlagChange__Setup(t *testing.T) {
	trigger := &OnFeatureFlagChange{}

	t.Run("at least one event required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnFeatureFlagChangeConfiguration{Events: []string{}},
		})

		require.ErrorContains(t, err, "at least one event type must be chosen")
	})

	t.Run("valid config requests webhook", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnFeatureFlagChangeConfiguration{Events: []string{KindFlag}},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		req, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok, "expected WebhookRequests[0] to be WebhookConfiguration")
		assert.Equal(t, []string{KindFlag}, req.Events)
	})

	t.Run("setup stores webhook URL in metadata", func(t *testing.T) {
		webhookCtx := &testWebhookContext{}
		metadataCtx := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Webhook:       webhookCtx,
			Metadata:      metadataCtx,
			Configuration: OnFeatureFlagChangeConfiguration{Events: []string{KindFlag}},
		})

		require.NoError(t, err)
		assert.True(t, webhookCtx.setupCalled)

		metadata, ok := metadataCtx.Metadata.(OnFeatureFlagChangeMetadata)
		require.True(t, ok)
		assert.Equal(t, "https://example.com/api/v1/webhooks/webhook-123", metadata.WebhookURL)
	})
}
