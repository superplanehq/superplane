package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GitLab__ListResources(t *testing.T) {
	g := &GitLab{}

	t.Run("returns empty list for unknown resource type", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{}
		resources, err := g.ListResources("unknown", core.ListResourcesContext{
			Integration: ctx,
		})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("returns list of repositories from metadata", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{
					{ID: 1, Name: "repo1", URL: "http://repo1"},
					{ID: 2, Name: "repo2", URL: "http://repo2"},
				},
			},
		}

		resources, err := g.ListResources("repository", core.ListResourcesContext{
			Integration: ctx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "repo1", resources[0].Name)
		assert.Equal(t, "repository", resources[0].Type)
		assert.Equal(t, "2", resources[1].ID)
		assert.Equal(t, "repo2", resources[1].Name)
	})

	t.Run("handles invalid metadata gracefully", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Metadata: "invalid-string-metadata",
		}

		_, err := g.ListResources("repository", core.ListResourcesContext{
			Integration: ctx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode metadata")
	})
}
