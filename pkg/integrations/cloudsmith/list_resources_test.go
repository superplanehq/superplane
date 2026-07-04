package cloudsmith

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

func okResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func Test__Cloudsmith__ListResources(t *testing.T) {
	integration := &Cloudsmith{}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := integration.ListResources("unknown", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("repository lists namespace-scoped resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"name": "Production", "slug": "production", "namespace": "acme"},
						{"name": "Staging", "slug": "staging", "namespace": "acme"}
					]`)),
				},
			},
		}

		resources, err := integration.ListResources("repository", core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "test-key",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "repository", resources[0].Type)
		assert.Equal(t, "acme/Production", resources[0].Name)
		assert.Equal(t, "acme/production", resources[0].ID)
		assert.Equal(t, "acme/staging", resources[1].ID)
	})

	t.Run("falls back to slug when name is empty", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"name": "", "slug": "production", "namespace": "acme"}
					]`)),
				},
			},
		}

		resources, err := integration.ListResources("repository", core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "test-key",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "acme/production", resources[0].Name)
	})

	t.Run("API error is propagated", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"detail":"server error"}`)),
				},
			},
		}

		_, err := integration.ListResources("repository", core.ListResourcesContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "test-key",
				},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing repositories")
	})
}
