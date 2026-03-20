package elastic

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func eq(value string) configuration.Predicate {
	return configuration.Predicate{Type: configuration.PredicateTypeEquals, Value: value}
}

func Test__OnAlertFires__Configuration(t *testing.T) {
	fields := (&OnAlertFires{}).Configuration()

	var statusesField *configuration.Field
	for i := range fields {
		if fields[i].Name == "statuses" {
			statusesField = &fields[i]
			break
		}
	}

	require.NotNil(t, statusesField)
	assert.Equal(t, configuration.FieldTypeAnyPredicateList, statusesField.Type)
	assert.Equal(t, []map[string]any{
		{"type": configuration.PredicateTypeEquals, "value": "active"},
	}, statusesField.Default)
}

func Test__OnAlertFires__HandleWebhook(t *testing.T) {
	trigger := &OnAlertFires{}
	secret := "auto-generated-secret"
	webhook := &contexts.NodeWebhookContext{Secret: secret}

	validBody := []byte(`{
		"eventType": "alert_fired",
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
		eventData := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "rule-123", eventData["ruleId"])
		assert.Equal(t, "High error rate", eventData["ruleName"])
		assert.NotContains(t, eventData, "payload")
	})

	t.Run("wrong eventType -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"eventType":"document_indexed","index":"my-index"}`),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	// --- rules ---

	t.Run("rules matches by rule ID -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"rules": []string{"rule-123", "rule-456"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("rules matches by rule name -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"rules": []string{"High error rate"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("rules does not match -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"rules": []string{"rule-999"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	// --- spaces ---

	t.Run("spaces matches -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"spaces": []string{"default"}},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("spaces does not match -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"spaces": []string{"production"}},
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
			Configuration: map[string]any{"tags": []configuration.Predicate{eq("env:prod")}},
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
			Configuration: map[string]any{"tags": []configuration.Predicate{eq("env:staging")}},
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
			Configuration: map[string]any{"severities": []configuration.Predicate{eq("critical"), eq("high")}},
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
			Configuration: map[string]any{"severities": []configuration.Predicate{eq("low")}},
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
			Configuration: map[string]any{"statuses": []configuration.Predicate{eq("active")}},
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
			Configuration: map[string]any{"statuses": []configuration.Predicate{eq("recovered")}},
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
				"rules":      []string{"rule-123"},
				"spaces":     []string{"default"},
				"tags":       []configuration.Predicate{eq("team:infra")},
				"severities": []configuration.Predicate{eq("critical")},
				"statuses":   []configuration.Predicate{eq("active")},
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
				"rules":    []string{"rule-123"},
				"statuses": []configuration.Predicate{eq("recovered")}, // active != recovered
			},
			Events:  eventsCtx,
			Webhook: webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})
}
