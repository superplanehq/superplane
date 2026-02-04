package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetRelease__Execute__Validation(t *testing.T) {
	component := GetRelease{}
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}

	t.Run("returns error when releaseId is nil for byId strategy", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				InstallationID: "12345",
				Owner:          "testhq",
				Repositories:   []Repository{helloRepo},
				GitHubApp:      GitHubAppMetadata{ID: 12345},
			},
			Configuration: map[string]any{
				"privateKey": "test-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{Metadata: NodeMetadata{Repository: &helloRepo}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":      "hello",
				"releaseStrategy": "byId",
				"releaseId":       nil,
			},
		})

		require.ErrorContains(t, err, "release ID is required when using byId strategy")
	})

	t.Run("returns error when tagName is empty for specific strategy", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				InstallationID: "12345",
				Owner:          "testhq",
				Repositories:   []Repository{helloRepo},
				GitHubApp:      GitHubAppMetadata{ID: 12345},
			},
			Configuration: map[string]any{
				"privateKey": "test-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{Metadata: NodeMetadata{Repository: &helloRepo}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":      "hello",
				"releaseStrategy": "specific",
				"tagName":         "",
			},
		})

		require.ErrorContains(t, err, "tag name is required when using specific tag strategy")
	})
}

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
