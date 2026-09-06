package opencodego

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func connectedIntegration(config map[string]any) *contexts.IntegrationContext {
	if config == nil {
		config = map[string]any{}
	}
	return &contexts.IntegrationContext{Configuration: config}
}

func syncContext(integration *contexts.IntegrationContext, httpContext *contexts.HTTPContext) core.SyncContext {
	return core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: integration.Configuration,
		BaseURL:       "https://app.superplane.test",
		HTTP:          httpContext,
		Integration:   integration,
	}
}

func Test__Sync(t *testing.T) {
	o := &OpenCodeGo{}

	t.Run("without an apiKey it fails", func(t *testing.T) {
		integration := connectedIntegration(map[string]any{})

		err := o.Sync(syncContext(integration, &contexts.HTTPContext{}))

		require.ErrorContains(t, err, "apiKey is required")
		assert.NotEqual(t, "ready", integration.State)
		assert.Empty(t, (&contexts.HTTPContext{}).Requests)
	})

	t.Run("with an apiKey it verifies and becomes ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"usage":{"rolling":{"status":"ok","percent":0,"resetsAt":"2026-08-22T23:29:53.432Z"}}}`),
			},
		}
		integration := connectedIntegration(map[string]any{"apiKey": "oc-test-key"})

		err := o.Sync(syncContext(integration, httpContext))

		require.NoError(t, err)
		assert.Equal(t, "ready", integration.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/usage")
		assert.Equal(t, "Bearer oc-test-key", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("a failed verification does not make the integration ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusUnauthorized, `{"type":"error","error":{"type":"AuthError","message":"Missing API key."}}`),
			},
		}
		integration := connectedIntegration(map[string]any{"apiKey": "oc-bad-key"})

		err := o.Sync(syncContext(integration, httpContext))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "AuthError")
		assert.Contains(t, err.Error(), "Missing API key.")
		assert.NotEqual(t, "ready", integration.State)
	})
}

func Test__ListResources__Models(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			response(http.StatusOK, `{"object":"list","data":[
				{"id":"glm-5.2","name":"GLM 5.2","owned_by":"opencode"},
				{"id":"kimi-k3","name":"Kimi K3","owned_by":"opencode"},
				{"id":"grok-4.5","name":"Grok 4.5","owned_by":"opencode"},
				{"id":"minimax-m3","name":"MiniMax M3","owned_by":"opencode"},
				{"id":"","name":"broken"}
			]}`),
		},
	}

	o := &OpenCodeGo{}
	resources, err := o.ListResources(ResourceTypeModel, core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        httpContext,
		Integration: connectedIntegration(map[string]any{"apiKey": "oc-test-key"}),
	})

	require.NoError(t, err)
	require.Len(t, resources, 4)
	assert.Equal(t, "glm-5.2", resources[0].ID)
	assert.Equal(t, "glm-5.2", resources[0].Name)
	assert.Equal(t, ResourceTypeModel, resources[0].Type)
	assert.Equal(t, "kimi-k3", resources[1].ID)
	assert.Equal(t, "grok-4.5", resources[2].ID)
	assert.Equal(t, "minimax-m3", resources[3].ID)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/models")
}

func Test__ListResources__UnknownType(t *testing.T) {
	o := &OpenCodeGo{}
	resources, err := o.ListResources("unknown", core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        &contexts.HTTPContext{},
		Integration: connectedIntegration(map[string]any{"apiKey": "oc-test-key"}),
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}

func Test__APIError(t *testing.T) {
	t.Run("parses anthropic-style errors with their type", func(t *testing.T) {
		err := apiError(http.StatusUnauthorized, []byte(`{"type":"error","error":{"type":"AuthError","message":"Missing API key."}}`))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "AuthError")
		assert.Contains(t, err.Error(), "Missing API key.")
	})

	t.Run("parses openai-style errors", func(t *testing.T) {
		err := apiError(http.StatusBadRequest, []byte(`{"error":{"message":"Model not found.","code":400}}`))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.Contains(t, err.Error(), "Model not found.")
	})

	t.Run("falls back to the raw body", func(t *testing.T) {
		err := apiError(http.StatusInternalServerError, []byte(`<html>boom</html>`))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
		assert.Contains(t, err.Error(), "<html>boom</html>")
	})
}
