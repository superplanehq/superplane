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

func Test__GetRecentWorkflowRuns__Setup(t *testing.T) {
	component := &GetRecentWorkflowRuns{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing project slug -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "",
				"workflowName": "build",
			},
		})

		require.ErrorContains(t, err, "project slug is required")
	})

	t.Run("missing workflow name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "",
			},
		})

		require.ErrorContains(t, err, "workflow name is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "build-test-deploy",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetRecentWorkflowRuns__Execute(t *testing.T) {
	component := &GetRecentWorkflowRuns{}

	t.Run("valid request -> emits individual workflow runs", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{
								"id": "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
								"branch": "main",
								"duration": 384,
								"created_at": "2021-09-01T22:49:03.544Z",
								"stopped_at": "2021-09-01T22:55:27.544Z",
								"credits_used": 150,
								"status": "success",
								"is_approval": false
							},
							{
								"id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
								"branch": "main",
								"duration": 412,
								"created_at": "2021-08-31T14:22:10.000Z",
								"stopped_at": "2021-08-31T14:29:02.000Z",
								"credits_used": 160,
								"status": "failed",
								"is_approval": false
							}
						],
						"next_page_token": ""
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
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "build-test-deploy",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, GetRecentWorkflowRunsPayloadType, execState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/insights/gh/acme/my-app/workflows/build-test-deploy")
		assert.NotContains(t, httpContext.Requests[0].URL.String(), "test-metrics")
	})

	t.Run("with branch filter -> sends branch query param", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [],
						"next_page_token": ""
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
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "build-test-deploy",
				"branch":       "feature/test",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "branch=feature%2Ftest")
	})

	t.Run("with special workflow name -> escapes workflow name in path", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [],
						"next_page_token": ""
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
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "build/test #1",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/workflows/build%2Ftest%20%231")
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
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
				"projectSlug":  "gh/acme/my-app",
				"workflowName": "nonexistent",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    integrationCtx,
		})

		require.ErrorContains(t, err, "failed to get recent workflow runs")
	})
}
