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
	assert.True(t, ruleField.Required)
	require.NotNil(t, ruleField.TypeOptions)
	require.NotNil(t, ruleField.TypeOptions.Resource)
	assert.Equal(t, ResourceTypeKibanaRule, ruleField.TypeOptions.Resource.Type)
	assert.False(t, ruleField.TypeOptions.Resource.Multi)

	var statusesField *configuration.Field
	for i := range fields {
		if fields[i].Name == "statuses" {
			statusesField = &fields[i]
			break
		}
	}

	require.NotNil(t, statusesField)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, statusesField.Type)
	require.NotNil(t, statusesField.TypeOptions)
	require.NotNil(t, statusesField.TypeOptions.Resource)
	assert.Equal(t, ResourceTypeKibanaAlertStatus, statusesField.TypeOptions.Resource.Type)
	assert.True(t, statusesField.TypeOptions.Resource.Multi)
}

func Test__OnAlertFires__Setup(t *testing.T) {
	trigger := &OnAlertFires{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"apiKey":    "api-key",
		},
	}

	t.Run("rules are required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{},
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "rule is required")
	})

	t.Run("valid rule and spaces are validated and stored in metadata", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"rule-123","name":"High error rate"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"default","name":"Default"}]`)),
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		testIntegration := &contexts.IntegrationContext{
			Configuration: integrationCtx.Configuration,
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"rule":   "rule-123",
				"spaces": []string{"default"},
			},
			Integration: testIntegration,
			HTTP:        httpCtx,
			Metadata:    metadataCtx,
			Requests:    requestCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, OnAlertFiresMetadata{
			RuleID:   "rule-123",
			RuleName: "High error rate",
			Spaces:   []string{"Default"},
		}, metadataCtx.Metadata)
		require.Len(t, testIntegration.WebhookRequests, 1)
		assert.Equal(t, map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		}, testIntegration.WebhookRequests[0])
		assert.Equal(t, checkAlertConnectorAction, requestCtx.Action)
	})

	t.Run("rule change -> stores previous rule ID in metadata", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"rule-456","name":"New rule"}`))},
			},
		}
		metadataCtx := &contexts.MetadataContext{Metadata: OnAlertFiresMetadata{
			RuleID:   "rule-123",
			RuleName: "Old rule",
		}}
		requestCtx := &contexts.RequestContext{}
		testIntegration := &contexts.IntegrationContext{Configuration: integrationCtx.Configuration}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"rule": "rule-456"},
			Integration:   testIntegration,
			HTTP:          httpCtx,
			Metadata:      metadataCtx,
			Requests:      requestCtx,
		})
		require.NoError(t, err)

		saved := metadataCtx.Metadata.(OnAlertFiresMetadata)
		assert.Equal(t, "rule-456", saved.RuleID)
		assert.Equal(t, "rule-123", saved.PreviousRuleID)
	})

	t.Run("unknown rule returns a validation error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"rule": "missing-rule",
			},
			Integration: integrationCtx,
			HTTP:        httpCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "failed to get Kibana rule")
	})
}

func Test__OnAlertFires__CheckConnectorAndAttachRule(t *testing.T) {
	trigger := &OnAlertFires{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"apiKey":    "api-key",
		},
	}
	const testWebhookURL = "https://superplane.test/api/v1/webhooks/test-123"
	connectorsResponse := `[{"id":"conn-1","name":"SuperPlane Alert","connector_type_id":".webhook","config":{"url":"` + testWebhookURL + `"}}]`
	ruleDetailsResponse := `{"id":"rule-123","name":"My rule","rule_type_id":".es-query","actions":[]}`
	ruleTypesResponse := `[{"id":".es-query","default_action_group_id":"query matched"}]`

	t.Run("no previous rule -> attaches connector and does not call detach", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			// FindKibanaWebhookConnector
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(connectorsResponse))},
			// EnsureKibanaRuleHasConnector: GetKibanaRule
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleDetailsResponse))},
			// EnsureKibanaRuleHasConnector: GetKibanaRuleDefaultActionGroupID
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleTypesResponse))},
			// EnsureKibanaRuleHasConnector: updateKibanaRule (attach)
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleDetailsResponse))},
		}}
		meta := &contexts.MetadataContext{Metadata: OnAlertFiresMetadata{RuleID: "rule-123"}}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:        checkAlertConnectorAction,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
			Requests:    &contexts.RequestContext{},
			Webhook:     &contexts.NodeWebhookContext{URL: testWebhookURL},
		})
		require.NoError(t, err)
		assert.Len(t, httpCtx.Requests, 4)
		saved := meta.Metadata.(OnAlertFiresMetadata)
		assert.Empty(t, saved.PreviousRuleID)
	})

	t.Run("previous rule set -> attaches to new rule and detaches from old", func(t *testing.T) {
		oldRuleDetailsResponse := `{"id":"rule-old","name":"Old rule","rule_type_id":".es-query","actions":[{"id":"conn-1","group":"query matched","params":{"body":"{}"}}]}`
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			// FindKibanaWebhookConnector
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(connectorsResponse))},
			// EnsureKibanaRuleHasConnector: GetKibanaRule
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleDetailsResponse))},
			// EnsureKibanaRuleHasConnector: GetKibanaRuleDefaultActionGroupID
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleTypesResponse))},
			// EnsureKibanaRuleHasConnector: updateKibanaRule (attach)
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleDetailsResponse))},
			// RemoveKibanaRuleConnector: GetKibanaRule (old rule)
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(oldRuleDetailsResponse))},
			// RemoveKibanaRuleConnector: updateKibanaRule (detach)
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(oldRuleDetailsResponse))},
		}}
		meta := &contexts.MetadataContext{Metadata: OnAlertFiresMetadata{
			RuleID:         "rule-123",
			PreviousRuleID: "rule-old",
		}}

		_, err := trigger.HandleAction(core.TriggerActionContext{
			Name:        checkAlertConnectorAction,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
			Requests:    &contexts.RequestContext{},
			Webhook:     &contexts.NodeWebhookContext{URL: testWebhookURL},
		})
		require.NoError(t, err)

		saved := meta.Metadata.(OnAlertFiresMetadata)
		assert.Empty(t, saved.PreviousRuleID)

		// Verify detach call targeted the old rule
		assert.Len(t, httpCtx.Requests, 6)
		assert.Contains(t, httpCtx.Requests[4].URL.Path, "rule-old")
	})
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
		"tags":     "team:infra,env:prod",
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
				"eventType": "alert_fired",
				"ruleName": "High error rate",
				"spaceId": "default",
				"tags": "team:infra,env:prod",
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
