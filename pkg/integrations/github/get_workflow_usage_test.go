package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetWorkflowUsage__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := GetWorkflowUsage{}

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

func Test__GetWorkflowUsage__Configuration(t *testing.T) {
	component := GetWorkflowUsage{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		require.GreaterOrEqual(t, len(config), 1)

		// Check repository field
		repoField := config[0]
		require.Equal(t, "repository", repoField.Name)
		require.True(t, repoField.Required)
	})

	t.Run("has optional workflow file field", func(t *testing.T) {
		var workflowField *struct {
			Name     string
			Required bool
		}
		for _, f := range config {
			if f.Name == "workflowFile" {
				workflowField = &struct {
					Name     string
					Required bool
				}{f.Name, f.Required}
				break
			}
		}
		require.NotNil(t, workflowField)
		require.False(t, workflowField.Required)
	})
}

func Test__GetWorkflowUsage__Metadata(t *testing.T) {
	component := GetWorkflowUsage{}

	t.Run("name is correct", func(t *testing.T) {
		require.Equal(t, "github.getWorkflowUsage", component.Name())
	})

	t.Run("label is correct", func(t *testing.T) {
		require.Equal(t, "Get Workflow Usage", component.Label())
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

	t.Run("icon is workflow", func(t *testing.T) {
		require.Equal(t, "workflow", component.Icon())
	})
}
