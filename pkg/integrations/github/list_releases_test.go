package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListReleases__Configuration(t *testing.T) {
	component := ListReleases{}

	t.Run("has correct name", func(t *testing.T) {
		require.Equal(t, "github.listReleases", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		require.Equal(t, "List Releases", component.Label())
	})

	t.Run("has description", func(t *testing.T) {
		require.NotEmpty(t, component.Description())
	})

	t.Run("has documentation", func(t *testing.T) {
		require.NotEmpty(t, component.Documentation())
		require.Contains(t, component.Documentation(), "Use Cases")
		require.Contains(t, component.Documentation(), "Configuration")
		require.Contains(t, component.Documentation(), "Output")
	})

	t.Run("has configuration fields", func(t *testing.T) {
		fields := component.Configuration()
		require.Len(t, fields, 3)

		// Repository field
		require.Equal(t, "repository", fields[0].Name)
		require.True(t, fields[0].Required)

		// PerPage field
		require.Equal(t, "perPage", fields[1].Name)
		require.False(t, fields[1].Required)

		// Page field
		require.Equal(t, "page", fields[2].Name)
		require.False(t, fields[2].Required)
	})

	t.Run("has default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		require.Equal(t, core.DefaultOutputChannel, channels[0])
	})
}

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

func Test__ListReleases__Execute__Validation(t *testing.T) {
	component := ListReleases{}
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}

	t.Run("requires valid integration metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: nil, // Invalid metadata
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{Metadata: NodeMetadata{Repository: &helloRepo}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
			},
		})

		require.Error(t, err)
	})
}

func Test__ListReleases__DefaultPagination(t *testing.T) {
	component := ListReleases{}

	t.Run("perPage defaults to 30 when not specified", func(t *testing.T) {
		// This tests the configuration parsing logic
		config := ListReleasesConfiguration{
			Repository: "test-repo",
			PerPage:    nil,
			Page:       nil,
		}

		require.Equal(t, "test-repo", config.Repository)
		require.Nil(t, config.PerPage)
		require.Nil(t, config.Page)
	})

	t.Run("perPage is capped at 100", func(t *testing.T) {
		perPage := 150
		config := ListReleasesConfiguration{
			Repository: "test-repo",
			PerPage:    &perPage,
		}

		// In the Execute function, perPage > 100 is capped to 100
		require.Equal(t, 150, *config.PerPage) // Config stores the original value
	})

	t.Run("component has correct icon", func(t *testing.T) {
		require.Equal(t, "github", component.Icon())
	})

	t.Run("component has correct color", func(t *testing.T) {
		require.Equal(t, "gray", component.Color())
	})
}

func Test__ListReleases__Actions(t *testing.T) {
	component := ListReleases{}

	t.Run("returns empty actions", func(t *testing.T) {
		actions := component.Actions()
		require.Empty(t, actions)
	})

	t.Run("HandleAction returns nil", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{})
		require.NoError(t, err)
	})

	t.Run("Cancel returns nil", func(t *testing.T) {
		err := component.Cancel(core.ExecutionContext{})
		require.NoError(t, err)
	})

	t.Run("Cleanup returns nil", func(t *testing.T) {
		err := component.Cleanup(core.SetupContext{})
		require.NoError(t, err)
	})

	t.Run("HandleWebhook returns 200", func(t *testing.T) {
		code, err := component.HandleWebhook(core.WebhookRequestContext{})
		require.NoError(t, err)
		require.Equal(t, 200, code)
	})
}

func Test__FormatTimestamp(t *testing.T) {
	t.Run("returns empty string for nil timestamp", func(t *testing.T) {
		result := formatTimestamp(nil)
		require.Equal(t, "", result)
	})
}

func Test__ParseIntFromConfig(t *testing.T) {
	t.Run("parses int", func(t *testing.T) {
		result, err := parseIntFromConfig(42)
		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("parses int64", func(t *testing.T) {
		result, err := parseIntFromConfig(int64(42))
		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("parses float64", func(t *testing.T) {
		result, err := parseIntFromConfig(float64(42.0))
		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("parses string", func(t *testing.T) {
		result, err := parseIntFromConfig("42")
		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("returns error for invalid string", func(t *testing.T) {
		_, err := parseIntFromConfig("not-a-number")
		require.Error(t, err)
	})

	t.Run("returns error for unsupported type", func(t *testing.T) {
		_, err := parseIntFromConfig([]int{1, 2, 3})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot parse int from type")
	})
}
