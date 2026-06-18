package cloudsmith

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// repoPage builds a JSON array of n repositories with sequential slugs (repo-1, repo-2, …).
func repoPage(n int) string {
	items := make([]string, n)
	for i := range items {
		items[i] = fmt.Sprintf(`{"name":"Repo %d","slug":"repo-%d","namespace":"acme"}`, i+1, i+1)
	}
	return "[" + strings.Join(items, ",") + "]"
}

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
				okResponse(`[
					{"name": "Production", "slug": "production", "namespace": "acme"},
					{"name": "Staging", "slug": "staging", "namespace": "acme"}
				]`),
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
				okResponse(`[{"name": "", "slug": "production", "namespace": "acme"}]`),
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

	t.Run("fetches all pages until partial page is returned", func(t *testing.T) {
		// First page is full (repositoryPageSize items); second page is partial.
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(repoPage(repositoryPageSize)),
				okResponse(repoPage(3)),
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
		assert.Len(t, resources, repositoryPageSize+3)
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

	t.Run("package lists packages in the repository parameter", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(`[
					{"name": "sp-compliance-mit", "version": "1.0.0", "slug_perm": "wxu9RDqPfCj0", "license": "MIT"},
					{"name": "sp-compliance-gpl", "version": "1.0.0", "slug_perm": "f3XvJCI9ufJa", "license": "GPL-3.0-only"}
				]`),
			},
		}

		resources, err := integration.ListResources("package", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
			Parameters:  map[string]string{"repository": "weskk/superplane-compliance"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "package", resources[0].Type)
		assert.Equal(t, "sp-compliance-mit 1.0.0 (MIT)", resources[0].Name)
		assert.Equal(t, "wxu9RDqPfCj0", resources[0].ID)
		assert.Equal(t, "f3XvJCI9ufJa", resources[1].ID)
	})

	t.Run("package returns empty when repository parameter is unset or an expression", func(t *testing.T) {
		for _, repo := range []string{"", "{{ $.trigger.data.repository }}"} {
			resources, err := integration.ListResources("package", core.ListResourcesContext{
				HTTP:        &contexts.HTTPContext{},
				Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "test-key"}},
				Parameters:  map[string]string{"repository": repo},
			})
			require.NoError(t, err)
			assert.Empty(t, resources)
		}
	})
}
