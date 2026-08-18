package openrouter

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

func Test__Sync(t *testing.T) {
	o := &OpenRouter{}

	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":{"label":"test","usage":0,"is_free_tier":true}}`),
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-or-v1-test"},
		}

		err := o.Sync(core.SyncContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{"apiKey": "sk-or-v1-test"},
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/key")
		assert.Equal(t, "Bearer sk-or-v1-test", httpContext.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, attributionTitle, httpContext.Requests[0].Header.Get("X-Title"))
		assert.Equal(t, attributionReferer, httpContext.Requests[0].Header.Get("HTTP-Referer"))
	})

	t.Run("invalid key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusUnauthorized, `{"error":{"message":"User not found.","code":401}}`),
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "bad"},
		}

		err := o.Sync(core.SyncContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{"apiKey": "bad"},
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "User not found.")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("missing api key", func(t *testing.T) {
		err := o.Sync(core.SyncContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{"apiKey": ""},
			HTTP:          &contexts.HTTPContext{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiKey is required")
	})

	t.Run("provisioning key failure does not block readiness", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":{"label":"test"}}`),
				response(http.StatusForbidden, `{"error":{"message":"Management key required.","code":403}}`),
			},
		}
		config := map[string]any{"apiKey": "sk-or-v1-test", "managementKey": "sk-or-provisioning"}
		integrationCtx := &contexts.IntegrationContext{Configuration: config}

		err := o.Sync(core.SyncContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: config,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/activity")
		assert.Equal(t, "Bearer sk-or-provisioning", httpContext.Requests[1].Header.Get("Authorization"))
	})
}

func Test__ListResources__Models(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			response(http.StatusOK, `{"data":[
				{"id":"openai/gpt-4o-mini","name":"OpenAI: GPT-4o-mini"},
				{"id":"anthropic/claude-sonnet-4.5","name":"Anthropic: Claude Sonnet 4.5"},
				{"id":"","name":"broken"}
			]}`),
		},
	}

	o := &OpenRouter{}
	resources, err := o.ListResources(ResourceTypeModel, core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        httpContext,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "openai/gpt-4o-mini", resources[0].ID)
	assert.Equal(t, "anthropic/claude-sonnet-4.5", resources[1].ID)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/models")
}

func Test__ListResources__Providers(t *testing.T) {
	t.Run("scoped to the selected model", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":{"id":"openai/gpt-4o-mini","endpoints":[
					{"provider_name":"Azure","tag":"azure"},
					{"provider_name":"OpenAI","tag":"openai"},
					{"provider_name":"Azure","tag":"azure/swedencentral"}
				]}}`),
			},
		}

		o := &OpenRouter{}
		resources, err := o.ListResources(ResourceTypeProvider, core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
			Parameters:  map[string]string{"model": "openai/gpt-4o-mini"},
		})

		require.NoError(t, err)

		// The regional Azure endpoint collapses into the azure provider slug,
		// which is the form routing accepts.
		require.Len(t, resources, 2)
		assert.Equal(t, "azure", resources[0].ID)
		assert.Equal(t, "Azure", resources[0].Name)
		assert.Equal(t, "openai", resources[1].ID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/models/openai/gpt-4o-mini/endpoints")
	})

	t.Run("falls back to every provider without a model", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":[
					{"name":"Cerebras","slug":"cerebras"},
					{"name":"Together","slug":"together"}
				]}`),
			},
		}

		o := &OpenRouter{}
		resources, err := o.ListResources(ResourceTypeProvider, core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "cerebras", resources[0].ID)
		assert.Equal(t, "Cerebras", resources[0].Name)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/providers")
	})
}

func Test__ListResources__UnknownType(t *testing.T) {
	o := &OpenRouter{}
	resources, err := o.ListResources("unknown", core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        &contexts.HTTPContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}
