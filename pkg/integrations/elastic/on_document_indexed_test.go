package elastic

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnDocumentIndexed__Setup(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"url":       "https://elastic.example.com",
		"kibanaUrl": "https://kibana.example.com",
		"authType":  "apiKey",
		"apiKey":    "test-api-key",
	}}

	t.Run("missing index -> error", func(t *testing.T) {
		err := (&OnDocumentIndexed{}).Setup(core.TriggerContext{
			Configuration: map[string]any{},
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "index is required")
	})

	t.Run("valid config -> schedules checkConnectorAvailability", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"index":"my-index"}]`)),
				},
			},
		}
		err := (&OnDocumentIndexed{}).Setup(core.TriggerContext{
			Configuration: map[string]any{"index": "my-index"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
			Metadata:      meta,
			Requests:      requests,
		})
		require.NoError(t, err)
		assert.Equal(t, checkConnectorAction, requests.Action)
		saved, ok := meta.Metadata.(OnDocumentIndexedMetadata)
		require.True(t, ok)
		assert.Equal(t, "my-index", saved.Index)
		assert.NotEmpty(t, saved.RouteKey)
		assert.NotEmpty(t, saved.LastTimestamp)
	})

	t.Run("rule already provisioned -> no-op", func(t *testing.T) {
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{
			LastTimestamp: "2024-06-01T12:00:00Z",
			RouteKey:      "route-123",
			RuleID:        "existing-rule-id",
		}}
		requests := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"index":"my-index"}]`)),
				},
			},
		}
		err := (&OnDocumentIndexed{}).Setup(core.TriggerContext{
			Configuration: map[string]any{"index": "my-index"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
			Metadata:      meta,
			Requests:      requests,
		})
		require.NoError(t, err)
		assert.Empty(t, requests.Action)
	})

	t.Run("index does not exist -> error", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"index":"other-index"}]`)),
				},
			},
		}
		err := (&OnDocumentIndexed{}).Setup(core.TriggerContext{
			Configuration: map[string]any{"index": "my-index"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
			Metadata:      meta,
			Requests:      requests,
		})
		require.ErrorContains(t, err, `index "my-index" was not found`)
	})
}

func Test__OnDocumentIndexed__CheckConnectorAndCreateRule(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"url":       "https://elastic.example.com",
		"kibanaUrl": "https://kibana.example.com",
		"authType":  "apiKey",
		"apiKey":    "test-api-key",
	}}

	connectorsResponse := `[{"id":"conn-123","name":"SuperPlane Alert"}]`
	ruleResponse := `{"id":"rule-456","name":"SuperPlane • my-index"}`

	t.Run("connector found -> creates rule and saves rule ID", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(connectorsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleResponse))},
			},
		}
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          checkConnectorAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		saved, ok := meta.Metadata.(OnDocumentIndexedMetadata)
		require.True(t, ok)
		assert.Equal(t, "rule-456", saved.RuleID)
		assert.NotEmpty(t, saved.RouteKey)
		assert.Empty(t, requests.Action)

		require.Len(t, httpCtx.Requests, 2)
		body, err := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		actions := payload["actions"].([]any)
		action := actions[0].(map[string]any)
		params := action["params"].(map[string]any)
		assert.Equal(t, onDocumentIndexedTimeField, payload["params"].(map[string]any)["timeField"])

		var actionBody map[string]any
		require.NoError(t, json.Unmarshal([]byte(params["body"].(string)), &actionBody))
		assert.Equal(t, "document_indexed", actionBody["eventType"])
		assert.Equal(t, saved.RouteKey, actionBody["routeKey"])
		assert.Equal(t, "my-index", actionBody["index"])
	})

	t.Run("connector not found yet -> reschedules", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			},
		}
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          checkConnectorAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Equal(t, checkConnectorAction, requests.Action)
	})

	t.Run("Kibana error listing connectors -> reschedules", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error":"internal"}`))},
			},
		}
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          checkConnectorAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Equal(t, checkConnectorAction, requests.Action)
	})

	t.Run("rule already provisioned -> no-op", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{RuleID: "existing-rule-id"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          checkConnectorAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Empty(t, requests.Action)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("uses correct Kibana URL and auth", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(connectorsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(ruleResponse))},
			},
		}
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          checkConnectorAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "kibana.example.com")
		assert.Equal(t, "ApiKey test-api-key", httpCtx.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.String(), "/api/alerting/rule")
	})
}

func Test__OnDocumentIndexed__HandleWebhook(t *testing.T) {
	trigger := &OnDocumentIndexed{}
	secret := "auto-generated-secret"
	webhook := &contexts.NodeWebhookContext{Secret: secret}

	validBody := []byte(`{"eventType":"document_indexed","routeKey":"route-123","index":"my-index"}`)

	headersWithSecret := func() http.Header {
		h := http.Header{}
		h.Set(SigningHeaderName, secret)
		return h
	}

	t.Run("secret missing -> 403", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: validBody, Headers: http.Header{},
			Configuration: map[string]any{}, Events: &contexts.EventContext{}, Webhook: webhook,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing required header")
	})

	t.Run("wrong secret -> 403", func(t *testing.T) {
		h := http.Header{}
		h.Set(SigningHeaderName, "wrong")
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: validBody, Headers: h,
			Configuration: map[string]any{}, Events: &contexts.EventContext{}, Webhook: webhook,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid value for header")
	})

	t.Run("invalid JSON -> 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte("not json"), Headers: headersWithSecret(),
			Configuration: map[string]any{}, Events: &contexts.EventContext{}, Webhook: webhook,
		})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "invalid JSON payload")
	})

	t.Run("wrong eventType -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"eventType":"alert_fired","routeKey":"route-123","index":"my-index"}`),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"index": "my-index"},
			Metadata:      &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{RouteKey: "route-123", LastTimestamp: "2024-06-01T12:00:00Z"}},
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"url": "https://elastic.example.com", "authType": "apiKey", "apiKey": "test-api-key"}},
			HTTP:          &contexts.HTTPContext{},
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	t.Run("routeKey mismatch -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"index": "my-index"},
			Metadata:      &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{RouteKey: "other-route", LastTimestamp: "2024-06-01T12:00:00Z"}},
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"url": "https://elastic.example.com", "authType": "apiKey", "apiKey": "test-api-key"}},
			HTTP:          &contexts.HTTPContext{},
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	t.Run("valid request -> queries Elasticsearch and emits documents", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"hits": {
							"hits": [
								{
									"_id": "doc-1",
									"_index": "my-index",
									"_source": {"@timestamp": "2024-06-01T12:01:00Z", "msg": "hello"}
								},
								{
									"_id": "doc-2",
									"_index": "my-index",
									"_source": {"@timestamp": "2024-06-01T12:02:00Z", "msg": "world"}
								}
							]
						}
					}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{
			LastTimestamp: "2024-06-01T12:00:00Z",
			RouteKey:      "route-123",
			RuleID:        "rule-456",
		}}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"index": "my-index"},
			Metadata:      meta,
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"url": "https://elastic.example.com", "authType": "apiKey", "apiKey": "test-api-key"}},
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, eventsCtx.Payloads, 2)
		assert.Equal(t, "elastic.document.indexed", eventsCtx.Payloads[0].Type)
		data := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "doc-1", data["id"])
		assert.Equal(t, "my-index", data["index"])

		saved, ok := meta.Metadata.(OnDocumentIndexedMetadata)
		require.True(t, ok)
		assert.Equal(t, "2024-06-01T12:02:00Z", saved.LastTimestamp)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://elastic.example.com/my-index/_search", httpCtx.Requests[0].URL.String())
	})

	t.Run("search failure -> returns 200 without emitting", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal"}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{
			LastTimestamp: "2024-06-01T12:00:00Z",
			RouteKey:      "route-123",
			RuleID:        "rule-456",
		}}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          validBody,
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"index": "my-index"},
			Metadata:      meta,
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"url": "https://elastic.example.com", "authType": "apiKey", "apiKey": "test-api-key"}},
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
		saved, ok := meta.Metadata.(OnDocumentIndexedMetadata)
		require.True(t, ok)
		assert.Equal(t, "2024-06-01T12:00:00Z", saved.LastTimestamp)
	})
}
