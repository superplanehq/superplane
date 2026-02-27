package launchdarkly

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

func Test__LaunchDarkly__Sync(t *testing.T) {
	i := &LaunchDarkly{}

	t.Run("no API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": ""},
		}

		err := i.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "API access token is required")
	})

	t.Run("successful sync -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"items":[{"key":"default","name":"Default Project"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "api-test-key"},
		}

		err := i.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://app.launchdarkly.com/api/v2/projects?limit=200&offset=0", httpContext.Requests[0].URL.String())
	})
}

func Test__LaunchDarkly__ListResources(t *testing.T) {
	i := &LaunchDarkly{}

	t.Run("unknown resource type -> empty", func(t *testing.T) {
		resources, err := i.ListResources("unknown", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("project -> list from API", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"items":[{"key":"default","name":"Default Project"},{"key":"mobile","name":"Mobile App"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}

		resources, err := i.ListResources("project", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "project", resources[0].Type)
		assert.Equal(t, "Default Project", resources[0].Name)
		assert.Equal(t, "default", resources[0].ID)
		assert.Equal(t, "Mobile App", resources[1].Name)
		assert.Equal(t, "mobile", resources[1].ID)
	})
}
