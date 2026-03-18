package elastic

import (
	"encoding/json"
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

var caseIntegrationCtx = &contexts.IntegrationContext{Configuration: map[string]any{
	"url":       "https://elastic.example.com",
	"kibanaUrl": "https://kibana.example.com",
	"authType":  "apiKey",
	"apiKey":    "test-api-key",
}}

const casesResponse = `{
	"cases": [
		{
			"id": "case-1",
			"title": "Production incident",
			"status": "in-progress",
			"severity": "high",
			"version": "WzE3LDFd",
			"tags": ["prod"],
			"description": "Error rate spike",
			"created_at": "2024-06-01T10:00:00.000Z",
			"updated_at": "2024-06-01T12:01:00.000Z"
		},
		{
			"id": "case-2",
			"title": "DB issue",
			"status": "closed",
			"severity": "low",
			"version": "WzE4LDFd",
			"tags": [],
			"description": "Resolved",
			"created_at": "2024-06-01T09:00:00.000Z",
			"updated_at": "2024-06-01T12:02:00.000Z"
		}
	]
}`

const caseWebhookSecret = "test-signing-secret"

func caseHeadersWithSecret() http.Header {
	h := http.Header{}
	h.Set(SigningHeaderName, caseWebhookSecret)
	return h
}

var caseWebhook = &contexts.NodeWebhookContext{Secret: caseWebhookSecret}

var caseStatusChangedBody = []byte(`{"eventType":"case_status_changed","routeKey":"route-123"}`)

func Test__OnCaseStatusChange__Setup(t *testing.T) {
	t.Run("new trigger -> initializes metadata, requests webhook, and schedules provisioning", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		}}
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := (&OnCaseStatusChange{}).Setup(core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Integration:   integration,
			Requests:      requests,
		})
		require.NoError(t, err)

		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, saved.LastPollTime)
		assert.NotEmpty(t, saved.RouteKey)
		assert.Empty(t, saved.RuleID)
		require.Len(t, integration.WebhookRequests, 1)
		cfg := integration.WebhookRequests[0].(map[string]any)
		assert.Equal(t, "https://kibana.example.com", cfg["kibanaUrl"])
		assert.Equal(t, checkCaseConnectorAction, requests.Action)
	})

	t.Run("re-save with existing rule -> keeps metadata and skips provisioning", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{
			LastPollTime: "2024-01-01T00:00:00Z",
			RouteKey:     "route-123",
			RuleID:       "existing-rule-id",
		}}
		requests := &contexts.RequestContext{}

		err := (&OnCaseStatusChange{}).Setup(core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Integration:   integration,
			Requests:      requests,
		})
		require.NoError(t, err)

		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.Equal(t, "2024-01-01T00:00:00Z", saved.LastPollTime)
		assert.Equal(t, "route-123", saved.RouteKey)
		assert.Equal(t, "existing-rule-id", saved.RuleID)
		require.Len(t, integration.WebhookRequests, 1)
		assert.Empty(t, requests.Action)
	})

	t.Run("cases configured -> resolves and stores case names", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"authType":  "apiKey",
			"apiKey":    "test-api-key",
		}}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"case-1","title":"Production incident","status":"open","severity":"high","version":"WzEsMV0="}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"case-2","title":"DB issue","status":"closed","severity":"low","version":"WzIsMV0="}`))},
			},
		}
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := (&OnCaseStatusChange{}).Setup(core.TriggerContext{
			Configuration: map[string]any{"cases": []string{"case-1", "case-2"}},
			Metadata:      meta,
			HTTP:          httpCtx,
			Integration:   integration,
			Requests:      requests,
		})
		require.NoError(t, err)

		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Production incident", saved.CaseNames["case-1"])
		assert.Equal(t, "DB issue", saved.CaseNames["case-2"])
		assert.NotEmpty(t, saved.RouteKey)
		assert.Equal(t, checkCaseConnectorAction, requests.Action)
	})
}

func Test__OnCaseStatusChange__CheckConnectorAndCreateRule(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"url":       "https://elastic.example.com",
		"kibanaUrl": "https://kibana.example.com",
		"authType":  "apiKey",
		"apiKey":    "test-api-key",
	}}

	connectorsResponse := `[{"id":"conn-123","name":"SuperPlane Alert"}]`
	ruleResponse := `{"id":"rule-456","name":"SuperPlane • Cases"}`

	t.Run("connector found -> creates rule and saves rule ID", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(connectorsResponse))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleResponse))},
		}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{RouteKey: "route-123", LastPollTime: "2024-06-01T12:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:        checkCaseConnectorAction,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
			Requests:    requests,
		})
		require.NoError(t, err)

		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.Equal(t, "rule-456", saved.RuleID)
		assert.Equal(t, "route-123", saved.RouteKey)
		assert.Empty(t, requests.Action)

		require.Len(t, httpCtx.Requests, 2)
		body, err := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		params := payload["params"].(map[string]any)
		assert.Equal(t, "cases.updated_at", params["timeField"])
		assert.Equal(t, []any{".kibana_alerting_cases"}, params["index"])

		actions := payload["actions"].([]any)
		action := actions[0].(map[string]any)
		bodyParams := action["params"].(map[string]any)
		var actionBody map[string]any
		require.NoError(t, json.Unmarshal([]byte(bodyParams["body"].(string)), &actionBody))
		assert.Equal(t, "case_status_changed", actionBody["eventType"])
		assert.Equal(t, "route-123", actionBody["routeKey"])
	})

	t.Run("connector not found yet -> reschedules", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
		}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{RouteKey: "route-123"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:        checkCaseConnectorAction,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
			Requests:    requests,
		})
		require.NoError(t, err)
		assert.Equal(t, checkCaseConnectorAction, requests.Action)
	})

	t.Run("Kibana error listing connectors -> reschedules", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error":"internal"}`))},
		}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{RouteKey: "route-123"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:        checkCaseConnectorAction,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
			Requests:    requests,
		})
		require.NoError(t, err)
		assert.Equal(t, checkCaseConnectorAction, requests.Action)
	})

	t.Run("rule already provisioned -> no-op", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{RuleID: "existing-rule-id", RouteKey: "route-123"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:        checkCaseConnectorAction,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    meta,
			Requests:    requests,
		})
		require.NoError(t, err)
		assert.Empty(t, requests.Action)
		assert.Empty(t, httpCtx.Requests)
	})
}

func Test__OnCaseStatusChange__HandleWebhook(t *testing.T) {
	t.Run("secret missing from request -> 403", func(t *testing.T) {
		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z", RouteKey: "route-123"}},
			Events:        &contexts.EventContext{},
			Webhook:       caseWebhook,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing required header")
	})

	t.Run("secret wrong value -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set(SigningHeaderName, "wrong-value")
		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       headers,
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z", RouteKey: "route-123"}},
			Events:        &contexts.EventContext{},
			Webhook:       caseWebhook,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid value for header")
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not json"),
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z", RouteKey: "route-123"}},
			Events:        &contexts.EventContext{},
			Webhook:       caseWebhook,
		})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "invalid JSON payload")
	})

	t.Run("wrong eventType -> silent pass", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"eventType":"document_indexed","routeKey":"route-123"}`),
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z", RouteKey: "route-123"}},
			Events:        events,
			Webhook:       caseWebhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("routeKey mismatch -> silent pass", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z", RouteKey: "other-route"}},
			Events:        events,
			Webhook:       caseWebhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("emits event for each updated case and advances checkpoint", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(casesResponse)),
		}}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z", RouteKey: "route-123"}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        events,
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 2, events.Count())
		assert.Equal(t, "elastic.case.status.changed", events.Payloads[0].Type)
		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.Equal(t, "2024-06-01T12:02:00.000Z", saved.LastPollTime)
	})

	t.Run("webhook uses Kibana URL and auth from integration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"cases":[]}`)),
		}}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z", RouteKey: "route-123"}}

		_, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        &contexts.EventContext{},
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Contains(t, req.URL.String(), "kibana.example.com")
		assert.Contains(t, req.URL.String(), "/api/cases")
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))
	})

	t.Run("caseId filter: only matching case emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(casesResponse)),
		}}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z", RouteKey: "route-123"}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{"cases": []string{"case-2"}},
			Metadata:      meta,
			Events:        events,
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "case-2", data["id"])
	})

	t.Run("status filter: only matching statuses emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(casesResponse)),
		}}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z", RouteKey: "route-123"}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{"statuses": []string{"closed"}},
			Metadata:      meta,
			Events:        events,
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "elastic.case.status.changed", events.Payloads[0].Type)
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "closed", data["status"])
	})

	t.Run("severity filter: only matching severities emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(casesResponse)),
		}}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z", RouteKey: "route-123"}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{"severities": []string{"high"}},
			Metadata:      meta,
			Events:        events,
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "high", data["severity"])
	})

	t.Run("tags filter: only cases with matching tag emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(casesResponse)),
		}}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z", RouteKey: "route-123"}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:    caseStatusChangedBody,
			Headers: caseHeadersWithSecret(),
			Configuration: map[string]any{
				"tags": []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "prod"}},
			},
			Metadata:    meta,
			Events:      events,
			Webhook:     caseWebhook,
			Integration: caseIntegrationCtx,
			HTTP:        httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "case-1", data["id"])
	})

	t.Run("no updated cases -> checkpoint unchanged", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"cases":[]}`)),
		}}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00Z", RouteKey: "route-123"}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        &contexts.EventContext{},
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.Equal(t, "2024-06-01T12:00:00Z", saved.LastPollTime)
	})

	t.Run("Kibana error -> returns 200 and leaves checkpoint unchanged", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{"error":"internal"}`)),
		}}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z", RouteKey: "route-123"}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        events,
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, events.Payloads)
		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.Equal(t, "2024-01-01T00:00:00Z", saved.LastPollTime)
	})
}

func Test__OnCaseStatusChange__Cleanup(t *testing.T) {
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}}}

	err := (&OnCaseStatusChange{}).Cleanup(core.TriggerContext{
		Metadata: &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{RuleID: "rule-456"}},
		HTTP:     httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"authType":  "apiKey",
			"apiKey":    "test-api-key",
		}},
	})
	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/alerting/rule/rule-456")
}
