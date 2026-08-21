package openrouter

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OpenRouter__Sync(t *testing.T) {
	o := &OpenRouter{}

	t.Run("success with api key -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"anthropic/claude-sonnet-4-6"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "sk-or-test",
			},
		}

		err := o.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://openrouter.ai/api/v1/models", httpContext.Requests[0].URL.String())
	})

	t.Run("missing api key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		err := o.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiKey is required")
		assert.Empty(t, integrationCtx.State)
	})

	t.Run("custom base URL", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":  "sk-or-test",
				"baseURL": "https://openrouter.internal/api/v1",
			},
		}

		err := o.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://openrouter.internal/api/v1/models", httpContext.Requests[0].URL.String())
	})
}

func Test__OpenRouter__ListResources(t *testing.T) {
	o := &OpenRouter{}

	t.Run("lists models", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"anthropic/claude-sonnet-4-6"},{"id":""}]}`)),
				},
			},
		}

		resources, err := o.ListResources("model", core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "sk-or-test",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "anthropic/claude-sonnet-4-6", resources[0].ID)
		assert.Equal(t, "anthropic/claude-sonnet-4-6", resources[0].Name)
	})

	t.Run("unknown resource type is empty", func(t *testing.T) {
		resources, err := o.ListResources("unknown", core.ListResourcesContext{})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
