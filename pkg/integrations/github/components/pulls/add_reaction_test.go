package pulls

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__AddReaction__Setup(t *testing.T) {
	component := AddReaction{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"target": ReactionTargetIssueComment, "commentId": "42", "content": "eyes", "repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("comment ID is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"target": ReactionTargetIssueComment, "commentId": "", "content": "eyes", "repository": "hello"},
		})

		require.ErrorContains(t, err, "comment ID is required")
	})

	t.Run("reaction content is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"target": ReactionTargetIssueComment, "commentId": "42", "content": "", "repository": "hello"},
		})

		require.ErrorContains(t, err, "reaction content is required")
	})

	t.Run("target is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"target": "", "commentId": "42", "content": "eyes", "repository": "hello"},
		})

		require.ErrorContains(t, err, "target is required")
	})

	t.Run("invalid target", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"target": "issue", "commentId": "42", "content": "eyes", "repository": "hello"},
		})

		require.ErrorContains(t, err, "invalid target")
	})
}

func Test__AddReaction__Execute(t *testing.T) {
	component := AddReaction{}

	t.Run("fails when target is invalid", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"target":     "invalid",
				"commentId":  "42",
				"content":    "eyes",
				"repository": "hello",
			},
		})

		require.ErrorContains(t, err, "invalid target")
	})

	t.Run("fails when comment ID is not a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"target":     ReactionTargetIssueComment,
				"commentId":  "abc",
				"content":    "eyes",
				"repository": "hello",
			},
		})

		require.ErrorContains(t, err, "comment ID is not a number")
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

func Test__parseCommentID(t *testing.T) {
	t.Run("parses regular integer string", func(t *testing.T) {
		commentID, err := parseCommentID("3983993590")
		require.NoError(t, err)
		require.EqualValues(t, 3983993590, commentID)
	})

	t.Run("parses scientific notation string", func(t *testing.T) {
		commentID, err := parseCommentID("3.98399359e+09")
		require.NoError(t, err)
		require.EqualValues(t, 3983993590, commentID)
	})

	t.Run("rejects decimal value", func(t *testing.T) {
		_, err := parseCommentID("3983993590.5")
		require.ErrorContains(t, err, "value has decimals")
	})
}
