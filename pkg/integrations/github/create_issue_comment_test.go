package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateIssueComment__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := CreateIssueComment{}

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

func Test__CreateIssueComment__Configuration(t *testing.T) {
	component := CreateIssueComment{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		fieldNames := make([]string, len(config))
		for i, field := range config {
			fieldNames[i] = field.Name
		}

		require.Contains(t, fieldNames, "repository")
		require.Contains(t, fieldNames, "issueNumber")
		require.Contains(t, fieldNames, "body")
	})

	t.Run("repository field is required", func(t *testing.T) {
		for _, field := range config {
			if field.Name == "repository" {
				require.True(t, field.Required)
			}
		}
	})

	t.Run("issueNumber field is required", func(t *testing.T) {
		for _, field := range config {
			if field.Name == "issueNumber" {
				require.True(t, field.Required)
			}
		}
	})

	t.Run("body field is required", func(t *testing.T) {
		for _, field := range config {
			if field.Name == "body" {
				require.True(t, field.Required)
			}
		}
	})
}

func Test__CreateIssueComment__Metadata(t *testing.T) {
	component := CreateIssueComment{}

	t.Run("has correct name", func(t *testing.T) {
		require.Equal(t, "github.createIssueComment", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		require.Equal(t, "Create Issue Comment", component.Label())
	})

	t.Run("has correct icon", func(t *testing.T) {
		require.Equal(t, "github", component.Icon())
	})

	t.Run("has correct color", func(t *testing.T) {
		require.Equal(t, "gray", component.Color())
	})

	t.Run("has default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		require.Equal(t, core.DefaultOutputChannel, channels[0])
	})

	t.Run("has description", func(t *testing.T) {
		require.NotEmpty(t, component.Description())
	})

	t.Run("has documentation", func(t *testing.T) {
		require.NotEmpty(t, component.Documentation())
	})
}
