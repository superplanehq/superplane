package circleci

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetWorkflow__Setup(t *testing.T) {
	component := &GetWorkflow{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing workflow ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workflowId": "",
			},
		})

		require.ErrorContains(t, err, "workflow ID is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workflowId": "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetWorkflow__Execute(t *testing.T) {
	component := &GetWorkflow{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("valid request -> emits workflow data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
						"name": "build-test-deploy",
						"status": "success",
						"created_at": "2021-09-01T22:49:03.544Z",
						"stopped_at": "2021-09-01T22:55:34.317Z"
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{
								"id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
								"name": "build",
								"type": "build",
								"status": "success",
								"started_at": "2021-09-01T22:49:05.000Z",
								"stopped_at": "2021-09-01T22:52:10.000Z",
								"job_number": 42,
								"project_slug": "gh/acme/my-app"
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"workflowId": "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, GetWorkflowPayloadType, execState.Type)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/workflow/fda08377-fe7e-46b1-8992-3a7aaecac9c3")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/workflow/fda08377-fe7e-46b1-8992-3a7aaecac9c3/job")
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "workflow not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"workflowId": "invalid-id",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "failed to get workflow")
	})
}
