package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListReleases__Name(t *testing.T) {
	component := &ListReleases{}
	assert.Equal(t, "github.listReleases", component.Name())
}

func Test__ListReleases__Label(t *testing.T) {
	component := &ListReleases{}
	assert.Equal(t, "List Releases", component.Label())
}

func Test__ListReleases__OutputChannels(t *testing.T) {
	component := &ListReleases{}
	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__ListReleases__Configuration(t *testing.T) {
	component := &ListReleases{}
	fields := component.Configuration()

	require.Len(t, fields, 3)

	// Repository field
	assert.Equal(t, "repository", fields[0].Name)
	assert.True(t, fields[0].Required)

	// PerPage field
	assert.Equal(t, "perPage", fields[1].Name)
	assert.False(t, fields[1].Required)

	// Page field
	assert.Equal(t, "page", fields[2].Name)
	assert.False(t, fields[2].Required)
}

func Test__ListReleases__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := &ListReleases{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
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
			Configuration: map[string]any{"repository": "inaccessible"},
		})

		require.ErrorContains(t, err, "repository inaccessible is not accessible to app installation")
	})

	t.Run("metadata is set successfully", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		metadata, ok := nodeMetadataCtx.Get().(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Repository)
		assert.Equal(t, "hello", metadata.Repository.Name)
	})
}
