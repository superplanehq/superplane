package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteAttachment__Setup(t *testing.T) {
	component := DeleteAttachment{}

	t.Run("missing attachment -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"issue": "ENG-142"},
		})

		require.ErrorContains(t, err, "attachment is required")
	})

	t.Run("blank attachment -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"attachment": "   "},
		})

		require.ErrorContains(t, err, "attachment is required")
	})

	t.Run("issue is optional", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Configuration: map[string]any{"attachment": "a1"},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteAttachment__Execute(t *testing.T) {
	component := DeleteAttachment{}

	t.Run("emits the deleted attachment id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentDelete":{"success":true}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  map[string]any{"issue": "ENG-142", "attachment": "a1"},
		})

		require.NoError(t, err)
		assert.Equal(t, AttachmentPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		result, ok := wrapped["data"].(*DeleteAttachmentResult)
		require.True(t, ok)
		assert.Equal(t, "a1", result.ID)
		assert.True(t, result.Deleted)

		assert.Equal(t, "a1", variablesFromRequest(t, httpContext, 0)["id"])
	})

	t.Run("an expression-supplied id needs no issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentDelete":{"success":true}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"attachment": "  a1  "},
		})

		require.NoError(t, err)
		assert.Equal(t, "a1", variablesFromRequest(t, httpContext, 0)["id"])
	})

	t.Run("blank attachment -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"attachment": "   "},
		})

		require.ErrorContains(t, err, "attachment is required")
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
			Configuration:  map[string]any{"attachment": "a1"},
		})

		require.ErrorContains(t, err, "Entity not found")
	})

	t.Run("unsuccessful response is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"attachmentDelete":{"success":false}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"attachment": "a1"},
		})

		require.ErrorContains(t, err, "not deleted")
	})
}
