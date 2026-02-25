package circleci

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetLastWorkflow__Setup(t *testing.T) {
	component := &GetLastWorkflow{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing project slug -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug": "",
			},
		})

		require.ErrorContains(t, err, "project slug is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug": "gh/acme/my-app",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with optional fields -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug": "gh/acme/my-app",
				"branch":      "main",
				"status":      "success",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetLastWorkflow__Execute(t *testing.T) {
	component := &GetLastWorkflow{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("valid request -> emits last workflow", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{
								"id": "1285fe1d-d3a6-44fc-8886-8979558254c4",
								"number": 130,
								"state": "created",
								"created_at": "2021-09-01T22:49:03.544Z"
							}
						]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{
								"id": "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
								"name": "build-test-deploy",
								"status": "success",
								"created_at": "2021-09-01T22:49:03.544Z",
								"stopped_at": "2021-09-01T22:55:34.317Z"
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
				"projectSlug": "gh/acme/my-app",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, GetLastWorkflowPayloadType, execState.Type)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/project/gh/acme/my-app/pipeline")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/pipeline/1285fe1d-d3a6-44fc-8886-8979558254c4/workflow")
	})

	t.Run("with status filter -> returns matching workflow", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "pipeline-1", "number": 130, "state": "created"}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{"id": "wf-1", "name": "build", "status": "failed"},
							{"id": "wf-2", "name": "deploy", "status": "success"}
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
				"projectSlug": "gh/acme/my-app",
				"status":      "success",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
	})

	t.Run("no matching workflow -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "pipeline-1", "number": 130, "state": "created"}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{"id": "wf-1", "name": "build", "status": "failed"}
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
				"projectSlug": "gh/acme/my-app",
				"status":      "success",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "no matching workflow found")
	})

	t.Run("status filter checks additional pipeline pages", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "pipeline-1", "number": 130, "state": "created"}],
						"next_page_token": "next-page-1"
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "wf-1", "name": "build", "status": "failed"}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "pipeline-2", "number": 129, "state": "created"}],
						"next_page_token": ""
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "wf-2", "name": "deploy", "status": "success"}]
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
				"projectSlug": "gh/acme/my-app",
				"status":      "success",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		require.Len(t, httpContext.Requests, 4)
		assert.Contains(t, httpContext.Requests[2].URL.String(), "page-token=next-page-1")
	})

	t.Run("repeated pagination token -> returns cycle detection error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "pipeline-1", "number": 130, "state": "created"}],
						"next_page_token": "repeat-token"
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "wf-1", "name": "build", "status": "failed"}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "pipeline-2", "number": 129, "state": "created"}],
						"next_page_token": "repeat-token"
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id": "wf-2", "name": "deploy", "status": "failed"}]
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
				"projectSlug": "gh/acme/my-app",
				"status":      "success",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "detected pagination cycle")
		require.Len(t, httpContext.Requests, 4)
	})

	t.Run("exceeded pagination page limit -> returns limit error", func(t *testing.T) {
		responses := make([]*http.Response, 0, maxGetLastWorkflowPipelinePages*2)
		for i := 1; i <= maxGetLastWorkflowPipelinePages; i++ {
			responses = append(responses, &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{
					"items": [{"id": "pipeline-%d", "number": %d, "state": "created"}],
					"next_page_token": "token-%d"
				}`, i, i, i))),
			})
			responses = append(responses, &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"items": [{"id": "wf-1", "name": "build", "status": "failed"}]
				}`)),
			})
		}

		httpContext := &contexts.HTTPContext{
			Responses: responses,
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug": "gh/acme/my-app",
				"status":      "success",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "exceeded maximum pipeline pages")
		require.Len(t, httpContext.Requests, maxGetLastWorkflowPipelinePages*2)
	})
}
