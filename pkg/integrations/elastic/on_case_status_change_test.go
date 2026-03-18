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

var caseStatusChangedBody = []byte(`{"eventType":"case_status_changed"}`)

func Test__OnCaseStatusChange__Setup(t *testing.T) {
	t.Run("new trigger -> initializes checkpoint and requests webhook", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		}}
		meta := &contexts.MetadataContext{}
		err := (&OnCaseStatusChange{}).Setup(core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Integration:   integration,
		})
		require.NoError(t, err)
		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, saved.LastPollTime)
		require.Len(t, integration.WebhookRequests, 1)
		cfg := integration.WebhookRequests[0].(map[string]any)
		assert.Equal(t, "https://kibana.example.com", cfg["kibanaUrl"])
	})

	t.Run("re-save preserves existing checkpoint", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z"}}
		err := (&OnCaseStatusChange{}).Setup(core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Integration:   integration,
		})
		require.NoError(t, err)
		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.Equal(t, "2024-01-01T00:00:00Z", saved.LastPollTime)
		require.Len(t, integration.WebhookRequests, 1)
	})
}

func Test__OnCaseStatusChange__HandleWebhook(t *testing.T) {
	t.Run("secret missing from request -> 403", func(t *testing.T) {
		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       http.Header{},
			Configuration: map[string]any{},
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
			Events:        &contexts.EventContext{},
			Webhook:       caseWebhook,
		})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "invalid JSON payload")
	})

	t.Run("wrong eventType -> silent pass", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"eventType":"alert_fired"}`),
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Events:        events,
			Webhook:       caseWebhook,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("emits event for each updated case and advances checkpoint", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(casesResponse))},
		}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z"}}

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
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cases":[]}`))},
		}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z"}}

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

	t.Run("status filter: only matching statuses emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(casesResponse))},
		}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z"}}

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
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(casesResponse))},
		}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z"}}

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
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(casesResponse))},
		}}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z"}}

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
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cases":[]}`))},
		}}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00Z"}}

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

	t.Run("Kibana error -> returns 500", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error":"internal"}`))},
		}}

		code, _, err := (&OnCaseStatusChange{}).HandleWebhook(core.WebhookRequestContext{
			Body:          caseStatusChangedBody,
			Headers:       caseHeadersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z"}},
			Events:        &contexts.EventContext{},
			Webhook:       caseWebhook,
			Integration:   caseIntegrationCtx,
			HTTP:          httpCtx,
		})
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.ErrorContains(t, err, "failed to list cases")
	})
}
