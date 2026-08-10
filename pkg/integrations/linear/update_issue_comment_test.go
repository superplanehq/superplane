package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssueComment__Setup(t *testing.T) {
	component := UpdateIssueComment{}

	t.Run("missing comment -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"body": "Updated"},
		})

		require.ErrorContains(t, err, "comment is required")
	})

	t.Run("blank comment -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"comment": "  ", "body": "Updated"},
		})

		require.ErrorContains(t, err, "comment is required")
	})

	t.Run("missing body -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"comment": "c1"},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("null body -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"comment": "c1", "body": nil},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("blank body -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"comment": "c1", "body": "   "},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"comment": "c1", "body": "Updated"},
		})

		require.NoError(t, err)
	})

	t.Run("issue is optional", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"issue": "ENG-142", "comment": "c1", "body": "Updated"},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateIssueComment__Execute(t *testing.T) {
	component := UpdateIssueComment{}

	t.Run("emits the updated comment", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"commentUpdate":{"success":true,"comment":{"id":"c1","body":"Updated","url":"https://linear.app/acme/issue/ENG-142#comment-c1","createdAt":"2026-03-27T16:42:11.123Z","updatedAt":"2026-03-27T18:05:47.482Z","editedAt":"2026-03-27T18:05:47.482Z","user":{"id":"u1","name":"Jane","displayName":"jane"},"issue":{"id":"i1","identifier":"ENG-142","title":"Boom","url":"https://linear.app/acme/issue/ENG-142"}}}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  map[string]any{"comment": "c1", "body": "Updated"},
		})

		require.NoError(t, err)
		assert.Equal(t, CommentPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		comment, ok := wrapped["data"].(*Comment)
		require.True(t, ok)
		assert.Equal(t, "c1", comment.ID)
		assert.Equal(t, "Updated", comment.Body)
		assert.Equal(t, "2026-03-27T18:05:47.482Z", comment.EditedAt)
		require.NotNil(t, comment.Issue)
		assert.Equal(t, "ENG-142", comment.Issue.Identifier)

		variables := variablesFromRequest(t, httpContext, 0)
		assert.Equal(t, "c1", variables["id"])
		input, ok := variables["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Updated", input["body"])
	})

	t.Run("trims the comment ID and body before sending", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"commentUpdate":{"success":true,"comment":{"id":"c1","body":"Updated","url":"https://linear.app/acme/issue/ENG-142#comment-c1"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"comment": "  c1  ", "body": "  Updated  "},
		})

		require.NoError(t, err)

		variables := variablesFromRequest(t, httpContext, 0)
		assert.Equal(t, "c1", variables["id"])
		input, ok := variables["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Updated", input["body"])
	})

	t.Run("missing comment -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"body": "Updated"},
		})

		require.ErrorContains(t, err, "comment is required")
	})

	t.Run("missing body -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"comment": "c1"},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("blank body -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"comment": "c1", "body": "   "},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("unsuccessful update -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"commentUpdate":{"success":false,"comment":null}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"comment": "c1", "body": "Updated"},
		})

		require.ErrorContains(t, err, "comment was not updated")
	})

	t.Run("API failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Entity not found"}]}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"comment": "c1", "body": "Updated"},
		})

		require.ErrorContains(t, err, "Entity not found")
	})
}
