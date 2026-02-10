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

func Test__SendLogEvent__Setup(t *testing.T) {
	component := SendLogEvent{}

	t.Run("requires at least one record", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"records": []map[string]any{},
			},
		})

		require.ErrorContains(t, err, "records is required")
	})

	t.Run("record message is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"records": []map[string]any{
					{
						"message": "   ",
					},
				},
			},
		})

		require.ErrorContains(t, err, "message is required")
	})
}

func Test__SendLogEvent__Execute(t *testing.T) {
	component := SendLogEvent{}

	t.Run("sends logs and emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"serviceName": "superplane.workflow",
				"records": []map[string]any{
					{
						"message":   "deployment completed",
						"severity":  "INFO",
						"timestamp": "2026-02-09T12:00:00Z",
						"attributes": map[string]any{
							"environment": "production",
						},
					},
				},
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, SendLogEventPayloadType, execCtx.Type)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/v1/logs")
		assert.Contains(t, httpContext.Requests[0].URL.Host, "ingress.")
		assert.Equal(t, "Bearer token123", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("invalid timestamp returns error", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"records": []map[string]any{
					{
						"message":   "deployment completed",
						"timestamp": "invalid-timestamp",
					},
				},
			},
			HTTP: &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.ErrorContains(t, err, "invalid timestamp")
	})

	t.Run("nil-like timestamp is treated as current time", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"records": []map[string]any{
					{
						"message":   "deployment completed",
						"timestamp": "<nil>",
					},
				},
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
	})
}
