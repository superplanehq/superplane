package issues

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RemoveIssueLabel__Setup(t *testing.T) {
	component := RemoveIssueLabel{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueNumber": "42", "labels": []string{"bug"}, "repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("issue number is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueNumber": "", "labels": []string{"bug"}, "repository": "hello"},
		})

		require.ErrorContains(t, err, "issue number is required")
	})

	t.Run("at least one label is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueNumber": "42", "labels": []string{}, "repository": "hello"},
		})

		require.ErrorContains(t, err, "at least one label is required")
	})
}

func Test__RemoveIssueLabel__Execute(t *testing.T) {
	component := RemoveIssueLabel{}

	t.Run("fails when issue number is not a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"issueNumber": "abc",
				"labels":      []string{"bug"},
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
