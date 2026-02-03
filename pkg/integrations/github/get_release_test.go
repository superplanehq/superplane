package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetRelease__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := GetRelease{}

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

func Test__GetRelease__Configuration(t *testing.T) {
	component := GetRelease{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		require.GreaterOrEqual(t, len(config), 2)

		// Check repository field
		repoField := config[0]
		require.Equal(t, "repository", repoField.Name)
		require.True(t, repoField.Required)

		// Check releaseStrategy field
		strategyField := config[1]
		require.Equal(t, "releaseStrategy", strategyField.Name)
		require.True(t, strategyField.Required)
	})

	t.Run("releaseStrategy has expected options", func(t *testing.T) {
		strategyField := config[1]
		require.NotNil(t, strategyField.TypeOptions)
		require.NotNil(t, strategyField.TypeOptions.Select)

		options := strategyField.TypeOptions.Select.Options
		require.GreaterOrEqual(t, len(options), 5)

		// Check that expected strategies are present
		optionValues := make([]string, len(options))
		for i, opt := range options {
			optionValues[i] = opt.Value
		}
		require.Contains(t, optionValues, "specific")
		require.Contains(t, optionValues, "byId")
		require.Contains(t, optionValues, "latest")
		require.Contains(t, optionValues, "latestDraft")
		require.Contains(t, optionValues, "latestPrerelease")
	})
}

func Test__GetRelease__Metadata(t *testing.T) {
	component := GetRelease{}

	t.Run("name is correct", func(t *testing.T) {
		require.Equal(t, "github.getRelease", component.Name())
	})

	t.Run("label is correct", func(t *testing.T) {
		require.Equal(t, "Get Release", component.Label())
	})

	t.Run("has description", func(t *testing.T) {
		require.NotEmpty(t, component.Description())
	})

	t.Run("has documentation", func(t *testing.T) {
		require.NotEmpty(t, component.Documentation())
	})

	t.Run("has default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		require.Equal(t, core.DefaultOutputChannel, channels[0])
	})
}
