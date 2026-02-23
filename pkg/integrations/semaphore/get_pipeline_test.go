package semaphore

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

func Test__GetPipeline__Setup(t *testing.T) {
	component := &GetPipeline{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing pipeline ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "",
			},
		})

		require.ErrorContains(t, err, "pipeline ID is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "00000000-0000-0000-0000-000000000000",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetPipeline__Execute(t *testing.T) {
	component := &GetPipeline{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("valid request -> emits pipeline data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipeline": {
							"name": "Initial Pipeline",
							"ppl_id": "00000000-0000-0000-0000-000000000000",
							"wf_id": "11111111-1111-1111-1111-111111111111",
							"state": "done",
							"result": "passed",
							"result_reason": "test",
							"branch_name": "main",
							"commit_sha": "a1b2c3d4e5f6",
							"yaml_file_name": "semaphore.yml",
							"working_directory": ".semaphore",
							"project_id": "22222222-2222-2222-2222-222222222222",
							"created_at": "2026-01-22T15:32:47Z",
							"done_at": "2026-01-22T15:32:56Z"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"pipelineId": "00000000-0000-0000-0000-000000000000",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/pipelines/00000000-0000-0000-0000-000000000000", httpContext.Requests[0].URL.String())
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "pipeline not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"pipelineId": "invalid-id",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "failed to get pipeline")
	})
}
