package grafana

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type testWebhookContext struct {
	id            string
	url           string
	configuration any
	metadata      any
	secret        []byte
}

func (t *testWebhookContext) GetID() string                 { return t.id }
func (t *testWebhookContext) GetURL() string                { return t.url }
func (t *testWebhookContext) GetSecret() ([]byte, error)    { return t.secret, nil }
func (t *testWebhookContext) GetMetadata() any              { return t.metadata }
func (t *testWebhookContext) GetConfiguration() any         { return t.configuration }
func (t *testWebhookContext) SetSecret(secret []byte) error { t.secret = secret; return nil }

func Test__GrafanaWebhookHandler__Setup__ProvisionContactPoint(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			// list contact points
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			// create contact point
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"uid":"cp_123","name":"ignored"}`))},
			// GET notification policies (for upsert)
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"receiver":"default"}`))},
			// PUT notification policies (for upsert)
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{}`))},
		},
	}

	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, metadata)

	result, ok := metadata.(GrafanaWebhookMetadata)
	require.True(t, ok)
	assert.Equal(t, "cp_123", result.ContactPointUID)
	assert.NotEmpty(t, result.ContactPointName)
	require.NotEmpty(t, webhookCtx.secret)

	require.Len(t, httpCtx.Requests, 4)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.String(), "/api/v1/provisioning/contact-points"))
	assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[1].URL.String(), "/api/v1/provisioning/contact-points"))
	assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[2].URL.String(), "/api/v1/provisioning/policies"))
	assert.Equal(t, http.MethodPut, httpCtx.Requests[3].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[3].URL.String(), "/api/v1/provisioning/policies"))

	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)

	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &payload))
	settings, ok := payload["settings"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://example.com/webhook", settings["url"])
	assert.Equal(t, "Bearer", settings["authorization_scheme"])
	assert.Equal(t, string(webhookCtx.secret), settings["authorization_credentials"])
}

func Test__GrafanaWebhookHandler__Setup__PolicyRouteHasAlertNameMatchers(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"uid":"cp_abc"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"receiver":"default"}`))},
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{}`))},
		},
	}
	webhookCtx := &testWebhookContext{
		id:  "wh_abc",
		url: "https://example.com/webhook",
		configuration: map[string]any{
			"alertNames": []any{
				map[string]any{"type": "equals", "value": "High CPU"},
				map[string]any{"type": "matches", "value": "Low.*"},
			},
		},
	}

	_, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"},
		},
	})
	require.NoError(t, err)

	// Verify the PUT policies body includes the alertname matcher.
	body, err := io.ReadAll(httpCtx.Requests[3].Body)
	require.NoError(t, err)
	var policyTree map[string]any
	require.NoError(t, json.Unmarshal(body, &policyTree))
	routes, ok := policyTree["routes"].([]any)
	require.True(t, ok)
	require.Len(t, routes, 1)
	route := routes[0].(map[string]any)
	matchers := route["object_matchers"].([]any)
	require.Len(t, matchers, 1)
	matcher := matchers[0].([]any)
	assert.Equal(t, "alertname", matcher[0])
	assert.Equal(t, "=~", matcher[1])
	// "High CPU" (equals) is quoted with QuoteMeta (space is not a metachar), "Low.*" (matches) is kept as-is
	assert.Equal(t, `High CPU|Low.*`, matcher[2])
}

func Test__GrafanaWebhookHandler__Setup__ManualFallbackWhenClientUnavailable(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, metadata)
	assert.NotEmpty(t, webhookCtx.secret)
}

func Test__GrafanaWebhookHandler__Setup__ManualFallbackOnNonRetriableProvisioningError(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`forbidden`)),
			},
		},
	}
	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, metadata)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
}

func Test__GrafanaWebhookHandler__Setup__RetriesOnRetriableProvisioningError(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`internal error`)),
			},
		},
	}
	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.ErrorContains(t, err, "will be retried")
	assert.Nil(t, metadata)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
}

func Test__GrafanaWebhookHandler__Setup__UsesLegacyConfiguredSecretWhenPresent(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"uid":"cp_legacy"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"receiver":"default"}`))},
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{}`))},
		},
	}

	webhookCtx := &testWebhookContext{
		id:            "wh_legacy",
		url:           "https://example.com/webhook",
		configuration: map[string]any{"sharedSecret": "legacy-secret"},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, metadata)
	assert.Equal(t, []byte("legacy-secret"), webhookCtx.secret)
}

func Test__GrafanaWebhookHandler__Cleanup(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	contactPointName := buildContactPointName("wh_123")
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			// GET policies (for route removal)
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"receiver":"default","routes":[{"receiver":"` + contactPointName + `","continue":true}]}`))},
			// PUT policies (with route removed)
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{}`))},
			// DELETE contact point
			{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(``))},
		},
	}
	webhookCtx := &testWebhookContext{
		id:       "wh_123",
		metadata: map[string]any{"contactPointUid": "cp_123", "contactPointName": contactPointName},
	}

	err := handler.Cleanup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 3)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.String(), "/api/v1/provisioning/policies"))
	assert.Equal(t, http.MethodPut, httpCtx.Requests[1].Method)
	assert.Equal(t, http.MethodDelete, httpCtx.Requests[2].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[2].URL.String(), "/api/v1/provisioning/contact-points/cp_123"))

	// Verify the PUT policies body has no routes (our route was removed).
	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)
	var policyTree map[string]any
	require.NoError(t, json.Unmarshal(body, &policyTree))
	routes, _ := policyTree["routes"].([]any)
	assert.Empty(t, routes)
}

func Test__GrafanaWebhookHandler__Cleanup__NoContactPointUIDWithoutTokenIsNoOp(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{}
	webhookCtx := &testWebhookContext{
		metadata: map[string]any{},
	}

	err := handler.Cleanup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 0)
}

func Test__GrafanaWebhookHandler__Cleanup__NilMetadataIsNoOp(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{}
	webhookCtx := &testWebhookContext{
		metadata: nil,
	}

	err := handler.Cleanup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 0)
}

func Test__GrafanaWebhookHandler__CompareConfig(t *testing.T) {
	handler := &GrafanaWebhookHandler{}

	equal, err := handler.CompareConfig(
		map[string]any{"webhookBindingKey": "node-1"},
		map[string]any{"webhookBindingKey": " node-1 "},
	)
	require.NoError(t, err)
	assert.True(t, equal)

	equal, err = handler.CompareConfig(
		map[string]any{"webhookBindingKey": "node-1", "sharedSecret": "secret-a"},
		map[string]any{"webhookBindingKey": "node-2", "sharedSecret": "secret-a"},
	)
	require.NoError(t, err)
	assert.False(t, equal)

	equal, err = handler.CompareConfig(
		map[string]any{
			"webhookBindingKey": "node-1",
			"sharedSecret":      "secret-a",
			"alertNames":        []any{map[string]any{"type": "equals", "value": "A"}},
		},
		map[string]any{
			"webhookBindingKey": "node-1",
			"sharedSecret":      "secret-a",
			"alertNames":        []any{map[string]any{"type": "equals", "value": "B"}},
		},
	)
	require.NoError(t, err)
	assert.False(t, equal)

	equal, err = handler.CompareConfig(
		map[string]any{"sharedSecret": "secret"},
		map[string]any{"sharedSecret": " secret "},
	)
	require.NoError(t, err)
	assert.True(t, equal)

	equal, err = handler.CompareConfig(
		map[string]any{
			"sharedSecret": "secret",
			"alertNames":   []any{map[string]any{"type": "equals", "value": "A"}},
		},
		map[string]any{
			"sharedSecret": "secret",
			"alertNames":   []any{map[string]any{"type": "equals", "value": "B"}},
		},
	)
	require.NoError(t, err)
	assert.False(t, equal)
}

func Test__GrafanaWebhookHandler__Merge(t *testing.T) {
	handler := &GrafanaWebhookHandler{}

	merged, changed, err := handler.Merge(
		map[string]any{"sharedSecret": "old", "webhookBindingKey": "node-1"},
		map[string]any{"sharedSecret": " new ", "webhookBindingKey": "node-1"},
	)
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, OnAlertFiringConfig{SharedSecret: "new", WebhookBindingKey: "node-1"}, merged)

	merged, changed, err = handler.Merge(
		map[string]any{"sharedSecret": " same ", "webhookBindingKey": "node-1"},
		map[string]any{"sharedSecret": "same", "webhookBindingKey": "node-1"},
	)
	require.NoError(t, err)
	require.False(t, changed)
	assert.Equal(t, OnAlertFiringConfig{SharedSecret: "same", WebhookBindingKey: "node-1"}, merged)

	merged, changed, err = handler.Merge(
		map[string]any{"sharedSecret": "keep-existing", "webhookBindingKey": "node-1"},
		map[string]any{},
	)
	require.NoError(t, err)
	require.False(t, changed)
	assert.Equal(t, OnAlertFiringConfig{SharedSecret: "keep-existing", WebhookBindingKey: "node-1"}, merged)

	mergedAny, changed, err := handler.Merge(
		map[string]any{
			"sharedSecret": "s", "webhookBindingKey": "node-1",
			"alertNames": []any{map[string]any{"type": "equals", "value": "A"}},
		},
		map[string]any{
			"sharedSecret": "s", "webhookBindingKey": "node-1",
			"alertNames": []any{map[string]any{"type": "equals", "value": "B"}},
		},
	)
	require.NoError(t, err)
	require.True(t, changed)
	mergedCfg := mergedAny.(OnAlertFiringConfig)
	require.Len(t, mergedCfg.AlertNames, 1)
	assert.Equal(t, "equals", mergedCfg.AlertNames[0].Type)
	assert.Equal(t, "B", mergedCfg.AlertNames[0].Value)

	mergedAny, changed, err = handler.Merge(
		map[string]any{
			"sharedSecret": "s", "webhookBindingKey": "node-1",
			"alertNames": []any{map[string]any{"type": "equals", "value": "Keep"}},
		},
		map[string]any{"sharedSecret": "s", "webhookBindingKey": "node-1"},
	)
	require.NoError(t, err)
	require.False(t, changed)
	mergedCfg = mergedAny.(OnAlertFiringConfig)
	require.Len(t, mergedCfg.AlertNames, 1)
	assert.Equal(t, "Keep", mergedCfg.AlertNames[0].Value)
}
