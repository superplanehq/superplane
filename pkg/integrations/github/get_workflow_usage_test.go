package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetWorkflowUsage__Setup(t *testing.T) {
	component := GetWorkflowUsage{}

	t.Run("fails when owner is not set", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				InstallationID: "12345",
				Owner:          "",
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "organization/owner is not set in integration metadata")
	})

	t.Run("fails when installation ID is not set", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Owner:          "test-org",
				InstallationID: "",
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "installation ID is not set in integration metadata")
	})

	t.Run("succeeds when metadata is valid", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Owner:          "test-org",
				InstallationID: "12345",
				GitHubApp: GitHubAppMetadata{
					ID:   123,
					Slug: "test-app",
				},
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
	})
}

func Test__GetWorkflowUsage__Execute(t *testing.T) {
	component := GetWorkflowUsage{}

	t.Run("fails when metadata decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{Metadata: "not a map"},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{},
		})

		require.ErrorContains(t, err, "failed to decode application metadata")
	})
}

func Test__GetWorkflowUsage__Configuration(t *testing.T) {
	component := GetWorkflowUsage{}

	t.Run("has no required configuration fields", func(t *testing.T) {
		config := component.Configuration()
		require.Empty(t, config, "GetWorkflowUsage should have no configuration fields since it uses org from integration")
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

	t.Run("description is set", func(t *testing.T) {
		require.NotEmpty(t, component.Description())
	})

	t.Run("documentation is set", func(t *testing.T) {
		require.NotEmpty(t, component.Documentation())
		require.Contains(t, component.Documentation(), "billable")
		require.Contains(t, component.Documentation(), "Administration")
	})

	t.Run("icon is github", func(t *testing.T) {
		require.Equal(t, "github", component.Icon())
	})

	t.Run("color is gray", func(t *testing.T) {
		require.Equal(t, "gray", component.Color())
	})

	t.Run("has default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		require.Equal(t, core.DefaultOutputChannel, channels[0])
	})
}
