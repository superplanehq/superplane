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
}

func Test__ListReleases__Configuration(t *testing.T) {
	component := ListReleases{}

	t.Run("returns correct configuration fields", func(t *testing.T) {
		config := component.Configuration()

		require.Len(t, config, 3)

		// Repository field
		require.Equal(t, "repository", config[0].Name)
		require.Equal(t, "Repository", config[0].Label)
		require.True(t, config[0].Required)

		// PerPage field
		require.Equal(t, "perPage", config[1].Name)
		require.Equal(t, "Per Page", config[1].Label)
		require.False(t, config[1].Required)
		require.Equal(t, 30, config[1].Default)

		// Page field
		require.Equal(t, "page", config[2].Name)
		require.Equal(t, "Page", config[2].Label)
		require.False(t, config[2].Required)
		require.Equal(t, 1, config[2].Default)
	})
}

func Test__ListReleases__Metadata(t *testing.T) {
	component := ListReleases{}

	t.Run("returns correct component name", func(t *testing.T) {
		require.Equal(t, "github.listReleases", component.Name())
	})

	t.Run("returns correct label", func(t *testing.T) {
		require.Equal(t, "List Releases", component.Label())
	})

	t.Run("returns correct description", func(t *testing.T) {
		require.Equal(t, "List releases for a GitHub repository", component.Description())
	})

	t.Run("returns documentation", func(t *testing.T) {
		doc := component.Documentation()
		require.Contains(t, doc, "List Releases")
		require.Contains(t, doc, "Use Cases")
		require.Contains(t, doc, "Configuration")
		require.Contains(t, doc, "Output")
	})

	t.Run("returns correct icon", func(t *testing.T) {
		require.Equal(t, "github", component.Icon())
	})

	t.Run("returns correct color", func(t *testing.T) {
		require.Equal(t, "gray", component.Color())
	})
}

func Test__ListReleases__OutputChannels(t *testing.T) {
	component := ListReleases{}

	t.Run("returns default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		require.Equal(t, core.DefaultOutputChannel, channels[0])
	})
}
