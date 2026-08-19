package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__AddIssueComment__Setup(t *testing.T) {
	component := AddIssueComment{}

	t.Run("missing issue -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"body": "Looks good"},
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("blank issue -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"issue": "  ", "body": "Looks good"},
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("missing body -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"issue": "ENG-142"},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("blank body -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"issue": "ENG-142", "body": "   "},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"issue": "ENG-142", "body": "Looks good"},
		})

		require.NoError(t, err)
	})
}

func Test__AddIssueComment__Execute(t *testing.T) {
	component := AddIssueComment{}

	t.Run("emits the created comment", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"commentCreate":{"success":true,"comment":{"id":"c1","body":"Looks good","url":"https://linear.app/acme/issue/ENG-142#comment-c1","user":{"id":"u1","name":"Jane","displayName":"jane"},"issue":{"id":"i1","identifier":"ENG-142","title":"Boom","url":"https://linear.app/acme/issue/ENG-142"}}}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  map[string]any{"issue": "ENG-142", "body": "Looks good"},
		})

		require.NoError(t, err)
		assert.Equal(t, CommentPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		comment, ok := wrapped["data"].(*Comment)
		require.True(t, ok)
		assert.Equal(t, "c1", comment.ID)
		assert.Equal(t, "https://linear.app/acme/issue/ENG-142#comment-c1", comment.URL)
		require.NotNil(t, comment.Issue)
		assert.Equal(t, "ENG-142", comment.Issue.Identifier)

		input, ok := variablesFromRequest(t, httpContext, 0)["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", input["issueId"])
		assert.Equal(t, "Looks good", input["body"])
	})

	t.Run("trims the body before sending", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"commentCreate":{"success":true,"comment":{"id":"c1","body":"Looks good","url":"https://linear.app/acme/issue/ENG-142#comment-c1"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"issue": "  ENG-142  ", "body": "  Looks good  "},
		})

		require.NoError(t, err)

		input, ok := variablesFromRequest(t, httpContext, 0)["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", input["issueId"])
		assert.Equal(t, "Looks good", input["body"])
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
			Configuration:  map[string]any{"issue": "ENG-142", "body": "Looks good"},
		})

		require.ErrorContains(t, err, "Entity not found")
	})
}
