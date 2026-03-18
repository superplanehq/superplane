package elastic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnCaseStatusChange__Setup(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"kibanaUrl": "https://kibana.example.com",
	}}

	t.Run("new trigger -> initializes LastPollTime, RouteKey, and requests webhook", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Integration:   integrationCtx,
		}
		err := (&OnCaseStatusChange{}).Setup(ctx)
		require.NoError(t, err)

		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.NotEmpty(t, saved.LastPollTime)
		assert.NotEmpty(t, saved.RouteKey)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})

	t.Run("re-save preserves existing LastPollTime and RouteKey", func(t *testing.T) {
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{
			LastPollTime: "2024-01-01T00:00:00Z",
			RouteKey:     "existing-route-key",
		}}
		integrationCtx2 := &contexts.IntegrationContext{Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		}}
		ctx := core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Integration:   integrationCtx2,
		}
		err := (&OnCaseStatusChange{}).Setup(ctx)
		require.NoError(t, err)

		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.Equal(t, "2024-01-01T00:00:00Z", saved.LastPollTime)
		assert.Equal(t, "existing-route-key", saved.RouteKey)
	})
}

func Test__OnCaseStatusChange__HandleWebhook(t *testing.T) {
	trigger := &OnCaseStatusChange{}
	secret := "test-webhook-secret"
	routeKey := "test-route-key"
	webhook := &contexts.NodeWebhookContext{Secret: secret}

	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"url":       "https://elastic.example.com",
		"kibanaUrl": "https://kibana.example.com",
		"authType":  "apiKey",
		"apiKey":    "test-api-key",
	}}

	casesResponse := `{
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

	headersWithSecret := func() http.Header {
		h := http.Header{}
		h.Set(SigningHeaderName, secret)
		return h
	}

	bodyWithRouteKey := func(routeKeyVal string) []byte {
		return []byte(`{"eventType":"case_status_changed","routeKey":"` + routeKeyVal + `"}`)
	}

	metaWithRouteKey := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{
			LastPollTime: "2024-06-01T12:00:00.000Z",
			RouteKey:     routeKey,
		}}
	}

	// --- auth ---

	t.Run("secret missing -> 403", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Metadata:      metaWithRouteKey(),
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
			Integration:   integrationCtx,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing required header")
	})

	t.Run("secret wrong value -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set(SigningHeaderName, "wrong-secret")
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headers,
			Configuration: map[string]any{},
			Metadata:      metaWithRouteKey(),
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
			Integration:   integrationCtx,
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid value for header")
	})

	t.Run("invalid JSON -> 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not json"),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      metaWithRouteKey(),
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
			Integration:   integrationCtx,
		})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "invalid JSON payload")
	})

	// --- routing ---

	t.Run("wrong eventType -> silent pass, no events", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"eventType":"alert_fired","routeKey":"` + routeKey + `"}`),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      metaWithRouteKey(),
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   integrationCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	t.Run("routeKey mismatch -> silent pass, no events", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey("different-route-key"),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      metaWithRouteKey(),
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   integrationCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	t.Run("empty routeKey in metadata -> silent pass", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{
			LastPollTime: "2024-06-01T12:00:00.000Z",
			RouteKey:     "",
		}}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   integrationCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	// --- checkpoint initialization ---

	t.Run("no LastPollTime -> initializes checkpoint, returns 200 with no events", func(t *testing.T) {
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{
			RouteKey: routeKey,
		}}
		eventsCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   integrationCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.NotEmpty(t, saved.LastPollTime)
	})

	// --- cases API + event emission ---

	t.Run("valid webhook -> emits event per updated case", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(casesResponse)),
				},
			},
		}
		eventsCtx := &contexts.EventContext{}
		meta := metaWithRouteKey()

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 2, eventsCtx.Count())
		assert.Equal(t, "elastic.case.status.changed", eventsCtx.Payloads[0].Type)
	})

	t.Run("status filter -> only matching cases emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(casesResponse)),
				},
			},
		}
		eventsCtx := &contexts.EventContext{}
		meta := metaWithRouteKey()

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{"statuses": []string{"closed"}},
			Metadata:      meta,
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, eventsCtx.Count())
		data := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "closed", data["status"])
	})

	t.Run("checkpoint advances to latest updated_at", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(casesResponse)),
				},
			},
		}
		meta := metaWithRouteKey()

		_, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.Equal(t, "2024-06-01T12:02:00.000Z", saved.LastPollTime)
	})

	t.Run("no cases returned -> checkpoint unchanged", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cases":[]}`)),
				},
			},
		}
		meta := metaWithRouteKey()
		origCheckpoint := meta.Metadata.(OnCaseStatusChangeMetadata).LastPollTime

		_, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      meta,
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.Equal(t, origCheckpoint, saved.LastPollTime)
	})

	t.Run("Kibana error -> 200 with no events", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal"}`)),
				},
			},
		}
		eventsCtx := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      metaWithRouteKey(),
			Events:        eventsCtx,
			Webhook:       webhook,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, eventsCtx.Payloads)
	})

	t.Run("webhook uses Kibana URL from integration config", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cases":[]}`)),
				},
			},
		}

		_, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          bodyWithRouteKey(routeKey),
			Headers:       headersWithSecret(),
			Configuration: map[string]any{},
			Metadata:      metaWithRouteKey(),
			Events:        &contexts.EventContext{},
			Webhook:       webhook,
			Integration:   integrationCtx,
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
}
