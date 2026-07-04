package pulls

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__CreatePullRequest__Setup(t *testing.T) {
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
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		require.NoError(t, component.Setup(core.SetupContext{
			Integration: integrationCtx,
			HTTP:        httpCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{
				"head": `{{$["github.onPush"].data.ref}}`,
				"base": `{{$["github.onPush"].data.ref}}`,
			}),
		}))
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
