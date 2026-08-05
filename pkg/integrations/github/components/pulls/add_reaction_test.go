package pulls

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
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

	t.Run("reaction content must be supported", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"target": ReactionTargetIssueComment, "commentId": "42", "content": "thumbs-up", "repository": "hello"},
		})

		require.ErrorContains(t, err, "invalid reaction content")
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

	for _, field := range []string{"repository", "commentId", "content"} {
		t.Run(field+" rejects whitespace-only values", func(t *testing.T) {
			configuration := map[string]any{
				"target":     ReactionTargetIssueComment,
				"commentId":  "42",
				"content":    "eyes",
				"repository": "hello",
			}
			configuration[field] = "   "

			err := component.Setup(core.SetupContext{
				Integration:   &contexts.IntegrationContext{},
				Metadata:      &contexts.MetadataContext{},
				Configuration: configuration,
			})

			require.ErrorContains(t, err, "required")
		})
	}
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

	t.Run("fails when comment ID is not positive", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"target":     ReactionTargetIssueComment,
				"commentId":  "0",
				"content":    "eyes",
				"repository": "hello",
			},
		})

		require.ErrorContains(t, err, "comment ID must be positive")
	})

	t.Run("fails before calling GitHub when reaction content is invalid", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"target":     ReactionTargetIssueComment,
				"commentId":  "42",
				"content":    "thumbs-up",
				"repository": "hello",
			},
		})

		require.ErrorContains(t, err, "invalid reaction content")
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	for _, tt := range []struct {
		name   string
		target string
		path   string
	}{
		{
			name:   "issue or PR conversation comment",
			target: ReactionTargetIssueComment,
			path:   "/repos/testhq/hello/issues/comments/42/reactions",
		},
		{
			name:   "inline PR review comment",
			target: ReactionTargetReviewComment,
			path:   "/repos/testhq/hello/pulls/comments/42/reactions",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{
					mocks.GitHubResponse(http.StatusCreated, `{"id":99,"content":"eyes"}`),
				},
			}
			state := &contexts.ExecutionStateContext{}

			err := component.Execute(core.ExecutionContext{
				Integration:    mocks.IntegrationContextForNewSetupFlow(),
				HTTP:           httpCtx,
				Metadata:       &contexts.MetadataContext{},
				Requests:       &contexts.RequestContext{},
				ExecutionState: state,
				Configuration: map[string]any{
					"target":     tt.target,
					"commentId":  "42",
					"content":    "eyes",
					"repository": "hello",
				},
			})

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			assert.Equal(t, tt.path, httpCtx.Requests[0].URL.Path)
			assert.True(t, state.Passed)
		})
	}

	t.Run("transient provider errors schedule a durable retry", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusServiceUnavailable, `{"message":"temporarily unavailable"}`),
			},
		}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"target":     ReactionTargetIssueComment,
				"commentId":  "42",
				"content":    "eyes",
				"repository": "hello",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, common.ReactionRetryHookName, requests.Action)
	})
}

func Test__AddReaction__HandleHookFinishesOnPreflightError(t *testing.T) {
	validConfiguration := map[string]any{
		"target":     ReactionTargetIssueComment,
		"commentId":  "42",
		"content":    "eyes",
		"repository": "hello",
	}

	for _, tt := range []struct {
		name          string
		configuration any
		invalidClient bool
		message       string
	}{
		{name: "configuration decode", configuration: "not a map", message: "failed to decode configuration"},
		{name: "target", configuration: map[string]any{"target": "invalid", "commentId": "42", "content": "eyes", "repository": "hello"}, message: "invalid target"},
		{name: "comment ID", configuration: map[string]any{"target": ReactionTargetIssueComment, "commentId": "abc", "content": "eyes", "repository": "hello"}, message: "comment ID is not a number"},
		{name: "reaction content", configuration: map[string]any{"target": ReactionTargetIssueComment, "commentId": "42", "content": "invalid", "repository": "hello"}, message: "invalid reaction content"},
		{name: "GitHub client", configuration: validConfiguration, invalidClient: true, message: "failed to initialize GitHub client"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			integration := mocks.IntegrationContextForNewSetupFlow()
			if tt.invalidClient {
				integration.CurrentSecrets = nil
			}
			state := &contexts.ExecutionStateContext{}

			err := (&AddReaction{}).HandleHook(core.ActionHookContext{
				Name:           common.ReactionRetryHookName,
				Integration:    integration,
				HTTP:           &contexts.HTTPContext{},
				Metadata:       &contexts.MetadataContext{},
				Requests:       &contexts.RequestContext{},
				ExecutionState: state,
				Configuration:  tt.configuration,
			})

			require.NoError(t, err)
			assert.True(t, state.Finished)
			assert.False(t, state.Passed)
			assert.Contains(t, state.FailureMessage, tt.message)
		})
	}
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
