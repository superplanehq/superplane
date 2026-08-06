package linear

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func reactionConfig() map[string]any {
	return map[string]any{"target": ReactionTargetIssue, "issue": "ENG-142", "emoji": "eyes"}
}

func reactionResponse() string {
	return `{"data":{"reactionCreate":{"success":true,"reaction":{"id":"r1","emoji":"eyes","createdAt":"2026-03-27T16:42:11.123Z","user":{"id":"u1","name":"Ada Lovelace","displayName":"ada"}}}}}`
}

func Test__AddReaction__Setup(t *testing.T) {
	component := AddReaction{}

	t.Run("missing issue for the issue target -> error", func(t *testing.T) {
		config := reactionConfig()
		delete(config, "issue")

		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: config,
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("missing comment for the comment target -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"target": ReactionTargetComment, "emoji": "eyes"},
		})

		require.ErrorContains(t, err, "comment is required")
	})

	t.Run("missing emoji -> error", func(t *testing.T) {
		config := reactionConfig()
		delete(config, "emoji")

		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: config,
		})

		require.ErrorContains(t, err, "emoji is required")
	})

	t.Run("invalid target -> error", func(t *testing.T) {
		config := reactionConfig()
		config["target"] = "project"

		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: config,
		})

		require.ErrorContains(t, err, "invalid target")
	})

	t.Run("an omitted target defaults to the issue", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"issue": "ENG-142", "emoji": "eyes"},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: reactionConfig(),
		})

		require.NoError(t, err)
	})
}

func Test__AddReaction__Execute(t *testing.T) {
	component := AddReaction{}

	t.Run("emits the created reaction", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(reactionResponse())},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  reactionConfig(),
		})

		require.NoError(t, err)
		assert.Equal(t, ReactionPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		reaction, ok := wrapped["data"].(*Reaction)
		require.True(t, ok)
		assert.Equal(t, "r1", reaction.ID)
		assert.Equal(t, "eyes", reaction.Emoji)

		input, ok := variablesFromRequest(t, httpContext, 0)["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", input["issueId"])
		assert.Equal(t, "eyes", input["emoji"])
		assert.NotContains(t, input, "commentId")
	})

	t.Run("reacts to a comment when that target is selected", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(reactionResponse())},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"target": ReactionTargetComment, "comment": "c1", "emoji": "tada"},
		})

		require.NoError(t, err)

		input, ok := variablesFromRequest(t, httpContext, 0)["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "c1", input["commentId"])
		assert.Equal(t, "tada", input["emoji"])
		assert.NotContains(t, input, "issueId")
	})

	//
	// Linear rejects an alias of an emoji already on the target, which means the
	// reaction is present - the desired end state, so re-runs must not fail.
	//
	t.Run("an already-present reaction is treated as success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"conflict on insert of Reaction"}]}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  reactionConfig(),
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		reaction, ok := wrapped["data"].(*Reaction)
		require.True(t, ok)
		assert.Equal(t, "eyes", reaction.Emoji)
	})

	t.Run("unrelated errors are surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(`{"errors":[{"message":"invalid emoji"}]}`)},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  reactionConfig(),
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.ErrorContains(t, err, "invalid emoji")
	})

	t.Run("blank issue -> error", func(t *testing.T) {
		config := reactionConfig()
		config["issue"] = "   "

		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
		})

		require.ErrorContains(t, err, "issue is required")
	})
}

// The picker offers the forms Linear stores, so a selected value can never be an
// alias of a present emoji - the one case that triggers a conflict.
func Test__AddReaction__EmojiOptionsAreStoredForms(t *testing.T) {
	component := AddReaction{}

	var options []configurationFieldOption
	for _, field := range component.Configuration() {
		if field.Name != "emoji" {
			continue
		}

		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Select)
		for _, option := range field.TypeOptions.Select.Options {
			options = append(options, configurationFieldOption{Label: option.Label, Value: option.Value})
		}
	}

	require.NotEmpty(t, options)
	for _, option := range options {
		assert.NotEqual(t, "thumbsup", option.Value, "thumbsup is an alias Linear stores as +1")
		assert.NotEqual(t, "thumbsdown", option.Value, "thumbsdown is an alias Linear stores as -1")
	}
}

type configurationFieldOption struct {
	Label string
	Value string
}
