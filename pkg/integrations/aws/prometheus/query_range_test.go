package prometheus

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

func Test__QueryRange__Setup(t *testing.T) {
	component := &QueryRange{}

	t.Run("missing step -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"query":     "up",
				"start":     "2026-06-08T09:00:00Z",
				"end":       "2026-06-08T10:00:00Z",
			},
		})

		require.ErrorContains(t, err, "step is required")
	})

	t.Run("valid configuration -> stores workspace alias in metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"workspace": {
							"alias": "metrics",
							"arn": "arn:aws:aps:us-east-1:123456789012:workspace/ws-abc123",
							"status": {"statusCode": "ACTIVE"},
							"workspaceId": "ws-abc123"
						}
					}`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"query":     "up",
				"start":     "2026-06-08T09:00:00Z",
				"end":       "2026-06-08T10:00:00Z",
				"step":      "1m",
			},
			HTTP:        httpContext,
			Integration: validIntegrationContext(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(WorkspaceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "metrics", stored.WorkspaceAlias)
	})
}

func Test__QueryRange__Execute(t *testing.T) {
	component := &QueryRange{}

	t.Run("valid request -> emits range query result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "success",
						"data": {
							"resultType": "matrix",
							"result": [
								{"metric": {}, "values": [[1717846800, "1"], [1717846860, "2"]]}
							]
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":                              "us-east-1",
				"workspace":                           "ws-abc123",
				"query":                               "sum(rate(http_requests_total[5m]))",
				"start":                               "2026-06-08T09:00:00Z",
				"end":                                 "2026-06-08T10:00:00Z",
				"step":                                "1m",
				"maxSamplesProcessedWarningThreshold": 1000,
				"maxSamplesProcessedErrorThreshold":   2000,
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		assert.Equal(t, "aws.prometheus.queryRange", execState.Type)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "matrix", payload["resultType"])
		result, ok := payload["result"].([]any)
		require.True(t, ok)
		assert.Len(t, result, 1)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "https://aps-workspaces.us-east-1.amazonaws.com/workspaces/ws-abc123/api/v1/query_range", request.URL.Scheme+"://"+request.URL.Host+request.URL.Path)
		assert.Equal(t, "sum(rate(http_requests_total[5m]))", request.URL.Query().Get("query"))
		assert.Equal(t, "2026-06-08T09:00:00Z", request.URL.Query().Get("start"))
		assert.Equal(t, "2026-06-08T10:00:00Z", request.URL.Query().Get("end"))
		assert.Equal(t, "1m", request.URL.Query().Get("step"))
		assert.Equal(t, "1000", request.URL.Query().Get("max_samples_processed_warning_threshold"))
		assert.Equal(t, "2000", request.URL.Query().Get("max_samples_processed_error_threshold"))
	})

	t.Run("Prometheus API status error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "error",
						"errorType": "bad_data",
						"error": "invalid range query"
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"workspace": "ws-abc123",
				"query":     "invalid{",
				"start":     "2026-06-08T09:00:00Z",
				"end":       "2026-06-08T10:00:00Z",
				"step":      "1m",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    validIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to execute PromQL query range")
		require.ErrorContains(t, err, "prometheus API error (bad_data): invalid range query")
		assert.False(t, execState.Finished)
		assert.Empty(t, execState.Payloads)
	})
}
