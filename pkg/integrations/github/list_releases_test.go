package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

		var nodeMetadata NodeMetadata
		nodeMetadata = nodeMetadataCtx.Get().(NodeMetadata)
		assert.Equal(t, helloRepo.ID, nodeMetadata.Repository.ID)
	})
}

func Test__ListReleases__Metadata(t *testing.T) {
	component := ListReleases{}

	t.Run("has correct name", func(t *testing.T) {
		assert.Equal(t, "github.listReleases", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		assert.Equal(t, "List Releases", component.Label())
	})

	t.Run("has description", func(t *testing.T) {
		assert.NotEmpty(t, component.Description())
	})

	t.Run("has documentation", func(t *testing.T) {
		docs := component.Documentation()
		assert.NotEmpty(t, docs)
		assert.Contains(t, docs, "List Releases")
		assert.Contains(t, docs, "pagination")
	})

	t.Run("has correct icon and color", func(t *testing.T) {
		assert.Equal(t, "github", component.Icon())
		assert.Equal(t, "gray", component.Color())
	})
}

func Test__ListReleases__OutputChannels(t *testing.T) {
	component := ListReleases{}
	channels := component.OutputChannels(nil)

	t.Run("has default output channel", func(t *testing.T) {
		assert.Len(t, channels, 1)
		assert.Equal(t, core.DefaultOutputChannel, channels[0])
	})
}

func Test__ListReleases__ExampleOutput(t *testing.T) {
	component := ListReleases{}
	output := component.ExampleOutput()

	t.Run("has default channel", func(t *testing.T) {
		assert.Contains(t, output, "default")
	})

	t.Run("example has releases", func(t *testing.T) {
		defaultOutput := output["default"].([]any)
		assert.Greater(t, len(defaultOutput), 0)
		
		firstRelease := defaultOutput[0].(map[string]any)
		assert.Contains(t, firstRelease, "id")
		assert.Contains(t, firstRelease, "tag_name")
		assert.Contains(t, firstRelease, "name")
		assert.Contains(t, firstRelease, "body")
		assert.Contains(t, firstRelease, "published_at")
		assert.Contains(t, firstRelease, "assets")
		assert.Contains(t, firstRelease, "tarball_url")
		assert.Contains(t, firstRelease, "zipball_url")
	})
}
