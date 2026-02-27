package launchdarkly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

var testLogger = logrus.NewEntry(logrus.New())

// hmacSignature computes the LaunchDarkly HMAC-SHA256 hex signature for a body.
func hmacSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func Test__OnFeatureFlagChange__HandleWebhook(t *testing.T) {
	trigger := &OnFeatureFlagChange{}

	// Default config: project only, accept all flags and environments
	defaultConfig := map[string]any{"projectKey": "default"}
	validSecret := "test-signing-secret"

	t.Run("missing signing secret -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: defaultConfig,
			Webhook:       &contexts.NodeWebhookContext{},
			Events:        &contexts.EventContext{},
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "signing secret is required")
	})

	t.Run("missing X-LD-Signature header -> 403", func(t *testing.T) {
		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          []byte(`{}`),
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing X-LD-Signature header")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"Test Flag"}`)
		headers := http.Header{}
		headers.Set("X-LD-Signature", "invalidsignature")

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("non-flag event kind -> no emit", func(t *testing.T) {
		body := []byte(`{"kind":"project","name":"Some Project"}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("no filters -> emit all flag events", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","titleVerb":"turned on the flag","title":"User turned on the flag My Feature","accesses":[{"action":"updateOn","resource":"proj/default:env/production:flag/my-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag.updateOn", eventContext.Payloads[0].Type)
		payload, ok := eventContext.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "flag", payload["kind"])
		assert.Equal(t, "My Feature", payload["name"])
	})

	t.Run("flag event without accesses -> emit with kind-only type", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"Simple Flag"}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag", eventContext.Payloads[0].Type)
	})

	t.Run("flag matches predicate -> emit", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Flag","accesses":[{"action":"updateOn","resource":"proj/default:env/production:flag/my-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectKey": "default",
				"flags":      []map[string]any{{"type": "equals", "value": "my-flag"}},
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
		body := []byte(`{"kind":"flag","name":"Other Flag","accesses":[{"action":"updateOn","resource":"proj/default:env/production:flag/other-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectKey": "default",
				"flags":      []map[string]any{{"type": "equals", "value": "my-flag"}},
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
		body := []byte(`{"kind":"flag","name":"Beta Flag","accesses":[{"action":"updateOn","resource":"proj/default:env/production:flag/beta-feature-123"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectKey": "default",
				"flags":      []map[string]any{{"type": "matches", "value": "^beta-.*"}},
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
		body := []byte(`{"kind":"flag","name":"My Flag","accesses":[{"action":"updateOn","resource":"proj/default:env/production:flag/my-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectKey":   "default",
				"environments": []string{"production"},
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
		body := []byte(`{"kind":"flag","name":"My Flag","accesses":[{"action":"updateOn","resource":"proj/default:env/development:flag/my-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectKey":   "default",
				"environments": []string{"production"},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("createFlag uses env wildcard and passes env filter -> emit", func(t *testing.T) {
		// LaunchDarkly uses proj/<proj>:env/*:flag/<flag> for createFlag (project-scoped).
		// The env wildcard should NOT be filtered out by an environment predicate.
		body := []byte(`{"kind":"flag","name":"New Flag","accesses":[{"action":"createFlag","resource":"proj/default:env/*:flag/new-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectKey":   "default",
				"environments": []string{"production"},
				"actions":      []string{ActionCreateFlag},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag.createFlag", eventContext.Payloads[0].Type)
	})

	t.Run("createFlag with flag predicate -> filters by flag key", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"Other Flag","accesses":[{"action":"createFlag","resource":"proj/default:env/*:flag/other-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"projectKey": "default",
				"flags":      []map[string]any{{"type": "equals", "value": "my-flag"}},
				"actions":    []string{ActionCreateFlag},
			},
			Webhook: wc,
			Events:  eventContext,
			Logger:  testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("action not in configured actions -> no emit", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","accesses":[{"action":"updateRules","resource":"proj/default:env/production:flag/my-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"projectKey": "default", "actions": []string{ActionUpdateOn}},
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("action in configured actions -> emit", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","accesses":[{"action":"updateOn","resource":"proj/default:env/production:flag/my-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"projectKey": "default", "actions": []string{ActionUpdateOn}},
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "launchdarkly.flag.updateOn", eventContext.Payloads[0].Type)
	})

	t.Run("empty actions config -> all actions accepted", func(t *testing.T) {
		body := []byte(`{"kind":"flag","name":"My Feature","accesses":[{"action":"deleteFlag","resource":"proj/default:env/production:flag/my-flag"}]}`)
		sig := hmacSignature(validSecret, body)
		headers := http.Header{}
		headers.Set("X-LD-Signature", sig)

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        eventContext,
			Logger:        testLogger,
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

		wc := &contexts.NodeWebhookContext{}
		require.NoError(t, wc.SetSecret([]byte(validSecret)))
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: defaultConfig,
			Webhook:       wc,
			Events:        &contexts.EventContext{},
			Logger:        testLogger,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing kind in payload")
	})
}

func Test__OnFeatureFlagChange__Setup(t *testing.T) {
	trigger := &OnFeatureFlagChange{}

	t.Run("missing project key -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: OnFeatureFlagChangeConfiguration{},
		})
		require.ErrorContains(t, err, "project key is required")
	})

	t.Run("project only requests webhook for all flags", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Webhook:       &contexts.NodeWebhookContext{},
			Configuration: OnFeatureFlagChangeConfiguration{ProjectKey: "default"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		req, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok, "expected WebhookRequests[0] to be WebhookConfiguration")
		assert.Equal(t, "default", req.ProjectKey)
	})

	t.Run("project with flags predicate requests webhook", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Webhook:     &contexts.NodeWebhookContext{},
			Configuration: OnFeatureFlagChangeConfiguration{
				ProjectKey: "default",
				Flags: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "my-flag"},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		req, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "default", req.ProjectKey)
	})
}
