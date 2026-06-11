package issues

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssueComment__Setup(t *testing.T) {
	component := UpdateIssueComment{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"commentId": "123", "body": "test", "repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("comment ID is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"commentId": "", "body": "test", "repository": "hello"},
		})

		require.ErrorContains(t, err, "comment ID is required")
	})

	t.Run("body is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"commentId": "123", "body": "", "repository": "hello"},
		})

		require.ErrorContains(t, err, "body is required")
	})
}

func Test__UpdateIssueComment__Execute(t *testing.T) {
	component := UpdateIssueComment{}

	t.Run("fails when comment ID is not a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"commentId":  "abc",
				"body":       "updated comment",
				"repository": "hello",
			},
		})

		require.ErrorContains(t, err, "comment ID is not a valid number")
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
