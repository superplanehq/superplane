package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__AddIssueAssignee__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := AddIssueAssignee{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueNumber": "42", "assignees": []string{"octocat"}, "repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("issue number is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueNumber": "", "assignees": []string{"octocat"}, "repository": "hello"},
		})

		require.ErrorContains(t, err, "issue number is required")
	})

	t.Run("at least one assignee is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueNumber": "42", "assignees": []string{}, "repository": "hello"},
		})

		require.ErrorContains(t, err, "at least one assignee is required")
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
			Configuration: map[string]any{"issueNumber": "42", "assignees": []string{"octocat"}, "repository": "world"},
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
			Configuration: map[string]any{"issueNumber": "42", "assignees": []string{"octocat"}, "repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
	})
}

func Test__AddIssueAssignee__Execute(t *testing.T) {
	component := AddIssueAssignee{}

	t.Run("fails when issue number is not a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"issueNumber": "abc",
				"assignees":   []string{"octocat"},
				"repository":  "hello",
			},
		})

		require.ErrorContains(t, err, "issue number is not a number")
	})

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}
