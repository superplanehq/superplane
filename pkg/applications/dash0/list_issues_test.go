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

	t.Run("valid setup", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{},
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
			Configuration: map[string]any{},
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
			Configuration: map[string]any{},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute Prometheus query")
	})

	t.Run("filters issues by check rules when specified", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First response: Prometheus query
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
											"check_rule": "rule-1",
											"issue_id": "issue-1"
										},
										"value": [1234567890, "1"]
									},
									{
										"metric": {
											"otel_metric_name": "dash0.issue.status",
											"check_rule": "rule-2",
											"issue_id": "issue-2"
										},
										"value": [1234567890, "1"]
									},
									{
										"metric": {
											"otel_metric_name": "dash0.issue.status",
											"check_rule": "rule-1",
											"issue_id": "issue-3"
										},
										"value": [1234567890, "1"]
									}
								]
							}
						}
					`)),
				},
				// Second response: List check rules
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						[
							{"id": "rule-1-id", "name": "rule-1"},
							{"id": "rule-2-id", "name": "rule-2"}
						]
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
				"checkRules": []string{"rule-1"},
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		require.Len(t, execCtx.Payloads, 1)

		// Extract the filtered results
		// Payload structure: {type, timestamp, data: {status, data: {resultType, result: [...]}}}
		payload := execCtx.Payloads[0].(map[string]any)
		responseData := payload["data"].(map[string]any)
		
		// The data field can be either a struct or map, handle both
		var results []any
		if dataSection, ok := responseData["data"].(map[string]any); ok {
			results = dataSection["result"].([]any)
		} else if dataSection, ok := responseData["data"].(PrometheusResponseData); ok {
			results = make([]any, len(dataSection.Result))
			for i, r := range dataSection.Result {
				results[i] = r
			}
		} else {
			t.Fatal("unable to extract results from payload")
		}
		
		// Should only have 2 results (issue-1 and issue-3, both with rule-1)
		assert.Len(t, results, 2)
	})

	t.Run("returns all issues when check rules are empty", func(t *testing.T) {
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
											"check_rule": "rule-1",
											"issue_id": "issue-1"
										},
										"value": [1234567890, "1"]
									},
									{
										"metric": {
											"otel_metric_name": "dash0.issue.status",
											"check_rule": "rule-2",
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
				"checkRules": []string{},
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		require.Len(t, execCtx.Payloads, 1)

		// Extract the results
		// Payload structure: {type, timestamp, data: {status, data: {resultType, result: [...]}}}
		payload := execCtx.Payloads[0].(map[string]any)
		responseData := payload["data"].(map[string]any)
		
		// The data field can be either a struct or map, handle both
		var results []any
		if dataSection, ok := responseData["data"].(map[string]any); ok {
			results = dataSection["result"].([]any)
		} else if dataSection, ok := responseData["data"].(PrometheusResponseData); ok {
			results = make([]any, len(dataSection.Result))
			for i, r := range dataSection.Result {
				results[i] = r
			}
		} else {
			t.Fatal("unable to extract results from payload")
		}
		
		// Should have all 2 results when check rules are empty
		assert.Len(t, results, 2)
	})
}
