package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListReleases__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := ListReleases{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set successfully", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
	})

	t.Run("accepts optional pagination parameters", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration: integrationCtx,
			Metadata:    &nodeMetadataCtx,
			Configuration: map[string]any{
				"repository": "hello",
				"perPage":    "50",
				"page":       "2",
			},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
	})
}

func Test__ListReleases__Configuration(t *testing.T) {
	component := ListReleases{}

	t.Run("has correct name", func(t *testing.T) {
		require.Equal(t, "github.listReleases", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		require.Equal(t, "List Releases", component.Label())
	})

	t.Run("has correct description", func(t *testing.T) {
		require.Equal(t, "List releases for a GitHub repository", component.Description())
	})

	t.Run("has three configuration fields", func(t *testing.T) {
		fields := component.Configuration()
		require.Len(t, fields, 3)

		// Repository field
		require.Equal(t, "repository", fields[0].Name)
		require.Equal(t, "Repository", fields[0].Label)
		require.True(t, fields[0].Required)

		// PerPage field
		require.Equal(t, "perPage", fields[1].Name)
		require.Equal(t, "Per Page", fields[1].Label)
		require.False(t, fields[1].Required)
		require.Equal(t, "30", fields[1].Default)

		// Page field
		require.Equal(t, "page", fields[2].Name)
		require.Equal(t, "Page", fields[2].Label)
		require.False(t, fields[2].Required)
		require.Equal(t, "1", fields[2].Default)
	})

	t.Run("has single default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		require.Equal(t, core.DefaultOutputChannel, channels[0])
	})

	t.Run("has github icon", func(t *testing.T) {
		require.Equal(t, "github", component.Icon())
	})

	t.Run("has gray color", func(t *testing.T) {
		require.Equal(t, "gray", component.Color())
	})

	t.Run("has no actions", func(t *testing.T) {
		require.Empty(t, component.Actions())
	})
}

func Test__ListReleases__Documentation(t *testing.T) {
	component := ListReleases{}

	t.Run("has documentation", func(t *testing.T) {
		doc := component.Documentation()
		require.NotEmpty(t, doc)
		require.Contains(t, doc, "List Releases")
		require.Contains(t, doc, "Use Cases")
		require.Contains(t, doc, "Configuration")
		require.Contains(t, doc, "Output")
	})
}
