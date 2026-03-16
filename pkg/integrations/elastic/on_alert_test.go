package elastic

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlertFires__HandleWebhook(t *testing.T) {
	trigger := &OnAlertFires{}
	secret := "auto-generated-secret"
	webhook := &contexts.NodeWebhookContext{Secret: secret}

	validBody := []byte(`{
		"ruleId":   "rule-123",
		"ruleName": "High error rate",
		"spaceId":  "default",
		"tags":     ["team:infra", "env:prod"],
		"status":   "active",
		"severity": "critical"
	}`)

	headersWithSecret := func() http.Header {
		h := http.Header{}
		h.Set(SigningHeaderName, secret)
		return h
	}

	t.Run("secret missing from request -> 403", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing required header")
	})

	t.Run("secret wrong value -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set(SigningHeaderName, "wrong-value")
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headers,
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid value for header")
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not json"),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
		})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "invalid JSON payload")
	})

	t.Run("no filters -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
		assert.Equal(t, "elastic.alert", eventsCtx.Payloads[0].Type)
	})

	// --- ruleIds ---

	t.Run("ruleIds matches -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"ruleIds": []string{"rule-123", "rule-456"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("ruleIds does not match -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"ruleIds": []string{"rule-999"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	// --- spaceIds ---

	t.Run("spaceIds matches -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"spaceIds": []string{"default"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("spaceIds does not match -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"spaceIds": []string{"production"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	// --- tags ---

	t.Run("tags any match -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"tags": []string{"env:prod"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("tags no match -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"tags": []string{"env:staging"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	// --- severities / statuses ---

	t.Run("severity filter matches -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"severities": []string{"critical", "high"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("severity filter does not match -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"severities": []string{"low"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	t.Run("status filter matches -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"statuses": []string{"active"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("status filter does not match -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"statuses": []string{"recovered"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	// --- combined filters ---

	t.Run("all filters match -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    validBody,
			Headers: headersWithSecret(),
			Configuration: map[string]any{
				"ruleIds":    []string{"rule-123"},
				"spaceIds":   []string{"default"},
				"tags":       []string{"team:infra"},
				"severities": []string{"critical"},
				"statuses":   []string{"active"},
			},
			Events:  eventsCtx,
			Webhook: webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("one filter mismatches -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    validBody,
			Headers: headersWithSecret(),
			Configuration: map[string]any{
				"ruleIds":  []string{"rule-123"},
				"statuses": []string{"recovered"}, // active != recovered
			},
			Events:  eventsCtx,
			Webhook: webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})
}
