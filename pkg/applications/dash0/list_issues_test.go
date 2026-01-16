package dash0

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

func Test__ListIssues__Setup(t *testing.T) {
	component := ListIssues{}

	t.Run("dataset is required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"dataset": ""},
		})

		require.ErrorContains(t, err, "dataset is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"dataset": "default",
			},
		})

		require.NoError(t, err)
	})
}

func Test__ListIssues__Execute(t *testing.T) {
	component := ListIssues{}

	t.Run("successful query returns issues", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {
											"otel_metric_name": "dash0.issue.status",
											"issue_id": "issue-1"
										},
										"value": [1234567890, "1"]
									},
									{
										"metric": {
											"otel_metric_name": "dash0.issue.status",
											"issue_id": "issue-2"
										},
										"value": [1234567890, "1"]
									}
								]
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataset": "default",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "dash0.issues.list", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)

		// Verify the request was made with the correct query
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
		assert.Equal(t, "Bearer token123", httpContext.Requests[0].Header.Get("Authorization"))
		
		// Verify the request body contains the correct query
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "query=%7Botel_metric_name%3D%22dash0.issue.status%22%7D+%3E%3D+1")
		assert.Contains(t, bodyStr, "dataset=default")
	})

	t.Run("query execution failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error","errorType":"bad_data","error":"parse error"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataset": "default",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute Prometheus query")
	})
}
