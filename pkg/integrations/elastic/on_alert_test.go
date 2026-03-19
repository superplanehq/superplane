package elastic

import (
	"io"
	"net/http"
	"strings"
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
	require.NotEmpty(t, fields)

	ruleField := fields[0]
	assert.Equal(t, "rule", ruleField.Name)
	assert.Equal(t, "Rule", ruleField.Label)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, ruleField.Type)
	assert.False(t, ruleField.Required)
	require.NotNil(t, ruleField.TypeOptions)
	require.NotNil(t, ruleField.TypeOptions.Resource)
	assert.Equal(t, ResourceTypeKibanaRule, ruleField.TypeOptions.Resource.Type)
	assert.False(t, ruleField.TypeOptions.Resource.Multi)
}

func Test__OnAlertFires__Setup__WithoutRule(t *testing.T) {
	trigger := &OnAlertFires{}
	httpCtx := &contexts.HTTPContext{}
	metadataCtx := &contexts.MetadataContext{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"apiKey":    "api-key",
		},
	}

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{},
		HTTP:          httpCtx,
		Metadata:      metadataCtx,
		Integration:   integrationCtx,
	})
	require.NoError(t, err)

	assert.Equal(t, OnAlertFiresMetadata{}, metadataCtx.Metadata)
	assert.Empty(t, httpCtx.Requests)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	assert.Equal(t, map[string]any{
		"kibanaUrl": "https://kibana.example.com",
		"ruleId":    "",
	}, integrationCtx.WebhookRequests[0])
}

func Test__OnAlertFires__Setup__WithRule(t *testing.T) {
	trigger := &OnAlertFires{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"rule-123","name":"High error rate"}`)),
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"apiKey":    "api-key",
		},
	}

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{"rule": "rule-123"},
		HTTP:          httpCtx,
		Metadata:      metadataCtx,
		Integration:   integrationCtx,
	})
	require.NoError(t, err)

	assert.Equal(t, OnAlertFiresMetadata{RuleName: "High error rate"}, metadataCtx.Metadata)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	assert.Equal(t, map[string]any{
		"kibanaUrl": "https://kibana.example.com",
		"ruleId":    "rule-123",
	}, integrationCtx.WebhookRequests[0])
}

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
		eventData := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "rule-123", eventData["ruleId"])
		assert.Equal(t, "High error rate", eventData["ruleName"])
		assert.NotContains(t, eventData, "payload")
	})

	// --- rule ---

	t.Run("selected rule matches by rule ID -> emits event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"rule": "rule-123"},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
	})

	t.Run("selected rule mismatch -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"rule": "rule-999"},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	t.Run("selected rule still emits when payload omits rule ID", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"ruleName": "High error rate",
				"spaceId": "default",
				"tags": ["team:infra", "env:prod"],
				"status": "active",
				"severity": "critical"
			}`),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"rule": "rule-123"},
			Events:        eventsCtx,
			Webhook:       webhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 1)
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
				"rule":       "rule-123",
				"spaces":     []string{"default"},
				"tags":       []configuration.Predicate{eq("team:infra")},
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
				"rule":     "rule-123",
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

func Test__Elastic__ListResources__AlertSeverityAndStatus(t *testing.T) {
	integration := &Elastic{}

	severities, err := integration.ListResources(ResourceTypeKibanaAlertSeverity, core.ListResourcesContext{})
	require.NoError(t, err)
	assert.Equal(t, []core.IntegrationResource{
		{Type: ResourceTypeKibanaAlertSeverity, ID: "low", Name: "Low"},
		{Type: ResourceTypeKibanaAlertSeverity, ID: "medium", Name: "Medium"},
		{Type: ResourceTypeKibanaAlertSeverity, ID: "high", Name: "High"},
		{Type: ResourceTypeKibanaAlertSeverity, ID: "critical", Name: "Critical"},
	}, severities)

	statuses, err := integration.ListResources(ResourceTypeKibanaAlertStatus, core.ListResourcesContext{})
	require.NoError(t, err)
	assert.Equal(t, []core.IntegrationResource{
		{Type: ResourceTypeKibanaAlertStatus, ID: "active", Name: "Active"},
		{Type: ResourceTypeKibanaAlertStatus, ID: "flapping", Name: "Flapping"},
		{Type: ResourceTypeKibanaAlertStatus, ID: "recovered", Name: "Recovered"},
		{Type: ResourceTypeKibanaAlertStatus, ID: "untracked", Name: "Untracked"},
	}, statuses)
}
