package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetRepositoryIssues__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := GetRepositoryIssues{}

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

func Test__GetRepositoryIssues__Configuration(t *testing.T) {
	component := GetRepositoryIssues{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		require.GreaterOrEqual(t, len(config), 2)

		// Check repository field
		repoField := config[0]
		require.Equal(t, "repository", repoField.Name)
		require.True(t, repoField.Required)

		// Check state field
		stateField := config[1]
		require.Equal(t, "state", stateField.Name)
		require.True(t, stateField.Required)
	})

	t.Run("state has expected options", func(t *testing.T) {
		stateField := config[1]
		require.NotNil(t, stateField.TypeOptions)
		require.NotNil(t, stateField.TypeOptions.Select)

		options := stateField.TypeOptions.Select.Options
		require.Len(t, options, 3)

		optionValues := make([]string, len(options))
		for i, opt := range options {
			optionValues[i] = opt.Value
		}
		require.Contains(t, optionValues, "open")
		require.Contains(t, optionValues, "closed")
		require.Contains(t, optionValues, "all")
	})

	t.Run("has labels field", func(t *testing.T) {
		var labelsField *struct {
			Name     string
			Required bool
		}
		for _, f := range config {
			if f.Name == "labels" {
				labelsField = &struct {
					Name     string
					Required bool
				}{f.Name, f.Required}
				break
			}
		}
		require.NotNil(t, labelsField)
		require.False(t, labelsField.Required)
	})

	t.Run("has sort options", func(t *testing.T) {
		var sortField *struct {
			Name    string
			Options []string
		}
		for _, f := range config {
			if f.Name == "sort" {
				options := make([]string, 0)
				if f.TypeOptions != nil && f.TypeOptions.Select != nil {
					for _, opt := range f.TypeOptions.Select.Options {
						options = append(options, opt.Value)
					}
				}
				sortField = &struct {
					Name    string
					Options []string
				}{f.Name, options}
				break
			}
		}
		require.NotNil(t, sortField)
		require.Contains(t, sortField.Options, "created")
		require.Contains(t, sortField.Options, "updated")
		require.Contains(t, sortField.Options, "comments")
	})
}

func Test__GetRepositoryIssues__Metadata(t *testing.T) {
	component := GetRepositoryIssues{}

	t.Run("name is correct", func(t *testing.T) {
		require.Equal(t, "github.getRepositoryIssues", component.Name())
	})

	t.Run("label is correct", func(t *testing.T) {
		require.Equal(t, "Get Repository Issues", component.Label())
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
