package linear

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetIssue__Setup(t *testing.T) {
	component := GetIssue{}

	t.Run("missing issue -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("blank issue -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issue": "   "},
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issue": "ENG-142"},
		})

		require.NoError(t, err)
	})
}

func Test__GetIssue__Execute(t *testing.T) {
	component := GetIssue{}

	t.Run("emits the fetched issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issue":{"id":"i1","identifier":"ENG-142","title":"Boom","url":"https://linear.app/acme/issue/ENG-142"}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  map[string]any{"issue": "ENG-142"},
		})

		require.NoError(t, err)
		assert.Equal(t, IssuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		issue, ok := wrapped["data"].(*Issue)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", issue.Identifier)
		assert.Equal(t, "https://linear.app/acme/issue/ENG-142", issue.URL)
	})

	t.Run("sends the issue id to Linear", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issue":{"id":"i1","identifier":"ENG-142"}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"issue": "  ENG-142  "},
		})

		require.NoError(t, err)

		variables := graphQLVariablesFromRequest(t, httpContext)
		assert.Equal(t, "ENG-142", variables["id"])
	})

	t.Run("missing issue -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"issue": "   "},
		})

		require.ErrorContains(t, err, "issue is required")
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
			Configuration:  map[string]any{"issue": "ENG-999"},
		})

		require.ErrorContains(t, err, "Entity not found")
	})
}

// graphQLVariablesFromRequest reads the `variables` object from the single
// GraphQL request captured by the HTTP context.
func graphQLVariablesFromRequest(t *testing.T, httpContext *contexts.HTTPContext) map[string]any {
	t.Helper()

	require.Len(t, httpContext.Requests, 1)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	payload := struct {
		Variables map[string]any `json:"variables"`
	}{}

	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.Variables
}
