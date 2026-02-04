package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateReview__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := CreateReview{}

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

func Test__CreateReview__Configuration(t *testing.T) {
	component := CreateReview{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		require.GreaterOrEqual(t, len(config), 3)

		// Check repository field
		repoField := config[0]
		require.Equal(t, "repository", repoField.Name)
		require.True(t, repoField.Required)

		// Check pull number field
		pullField := config[1]
		require.Equal(t, "pullNumber", pullField.Name)
		require.True(t, pullField.Required)

		// Check event field
		eventField := config[2]
		require.Equal(t, "event", eventField.Name)
		require.True(t, eventField.Required)
	})

	t.Run("has optional fields", func(t *testing.T) {
		var bodyField, commitField, commentsField *struct {
			Name     string
			Required bool
		}
		for _, f := range config {
			switch f.Name {
			case "body":
				bodyField = &struct {
					Name     string
					Required bool
				}{f.Name, f.Required}
			case "commitId":
				commitField = &struct {
					Name     string
					Required bool
				}{f.Name, f.Required}
			case "comments":
				commentsField = &struct {
					Name     string
					Required bool
				}{f.Name, f.Required}
			}
		}

		require.NotNil(t, bodyField)
		require.False(t, bodyField.Required)

		require.NotNil(t, commitField)
		require.False(t, commitField.Required)

		require.NotNil(t, commentsField)
		require.False(t, commentsField.Required)
	})
}

func Test__CreateReview__Metadata(t *testing.T) {
	component := CreateReview{}

	t.Run("name is correct", func(t *testing.T) {
		require.Equal(t, "github.createReview", component.Name())
	})

	t.Run("label is correct", func(t *testing.T) {
		require.Equal(t, "Create Review", component.Label())
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

	t.Run("icon is github", func(t *testing.T) {
		require.Equal(t, "github", component.Icon())
	})
}
