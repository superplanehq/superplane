package pulls

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreatePullRequest__Setup(t *testing.T) {
	helloRepo := common.Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := CreatePullRequest{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"head":       "feature",
			"base":       "main",
			"title":      "My PR",
		}
		for k, v := range overrides {
			config[k] = v
		}
		return config
	}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"repository": ""}),
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("head branch is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"head": ""}),
		})

		require.ErrorContains(t, err, "head branch is required")
	})

	t.Run("base branch is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"base": ""}),
		})

		require.ErrorContains(t, err, "base branch is required")
	})

	t.Run("title is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"title": ""}),
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("head and base must differ when both are literals", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"head": "main", "base": "main"}),
		})

		require.ErrorContains(t, err, "head and base branches must be different")
	})

	t.Run("head and base equality check is skipped when either is an expression", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo},
			},
		}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{
				"head": `{{$["github.onPush"].data.ref}}`,
				"base": `{{$["github.onPush"].data.ref}}`,
			}),
		}))
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo},
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"repository": "world"}),
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("repository expression skips setup validation", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo},
			},
		}
		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration: integrationCtx,
			Metadata:    &nodeMetadataCtx,
			Configuration: validConfig(map[string]any{
				"repository": `{{$["github.onPush"].data.repository.name}}`,
			}),
		}))
		require.Empty(t, nodeMetadataCtx.Get())
	})

	t.Run("metadata is set successfully", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				Repositories: []common.Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: validConfig(nil),
		}))

		require.Equal(t, nodeMetadataCtx.Get(), common.NodeMetadata{Repository: &helloRepo})
	})
}

func Test__CreatePullRequest__Execute(t *testing.T) {
	component := CreatePullRequest{}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("repository is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "",
				"head":       "feature",
				"base":       "main",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("head branch is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "",
				"base":       "main",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "head branch is required")
	})

	t.Run("base branch is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
				"base":       "",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "base branch is required")
	})

	t.Run("title is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
				"base":       "main",
				"title":      "",
			},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("head and base must differ", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "main",
				"base":       "main",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "head and base branches must be different")
	})
}
