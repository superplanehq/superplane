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

	t.Run("whitespace-only comment ID is rejected", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"commentId": "  ", "body": "test", "repository": "hello"},
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

	t.Run("handles scientific notation comment ID", func(t *testing.T) {
		// Large GitHub IDs can arrive as scientific notation through expressions
		id, err := parseCommentID("1.234567890e+09")
		require.NoError(t, err)
		require.Equal(t, int64(1234567890), id)
	})

	t.Run("handles plain integer comment ID", func(t *testing.T) {
		id, err := parseCommentID("1234567890")
		require.NoError(t, err)
		require.Equal(t, int64(1234567890), id)
	})

	t.Run("rejects NaN", func(t *testing.T) {
		_, err := parseCommentID("NaN")
		require.Error(t, err)
	})

	t.Run("rejects decimal", func(t *testing.T) {
		_, err := parseCommentID("123.456")
		require.Error(t, err)
	})

	t.Run("rejects empty", func(t *testing.T) {
		_, err := parseCommentID("")
		require.Error(t, err)
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
