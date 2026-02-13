package dash0

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetCheckDetails__Setup(t *testing.T) {
	component := GetCheckDetails{}

	t.Run("check id is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"checkId": "",
			},
		})

		require.ErrorContains(t, err, "checkId is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"checkId": "check-123",
			},
		})

		require.NoError(t, err)
	})

	t.Run("check id with only spaces is invalid", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"checkId": "   ",
			},
		})

		require.ErrorContains(t, err, "checkId is required")
	})
}

func Test__GetCheckDetails__Execute(t *testing.T) {
	component := GetCheckDetails{}

	t.Run("fetches details from failed-checks endpoint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"check-123","name":"Checkout latency"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "check-123",
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
		assert.Equal(t, GetCheckDetailsPayloadType, execCtx.Type)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/alerting/failed-checks/check-123")
	})

	t.Run("falls back to check-rules endpoint on 404", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"check-123","name":"Checkout latency rule"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "check-123",
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
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/alerting/failed-checks/check-123")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/api/alerting/check-rules/check-123")
	})

	t.Run("does not fallback when status is not 404", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"request got 404 code from downstream"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"check-123","name":"unexpected fallback"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "check-123",
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

		require.Error(t, err)
		assert.ErrorContains(t, err, "request got 500 code")
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/alerting/failed-checks/check-123")
	})

	t.Run("includeHistory adds include_history query parameter", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"check-123","name":"Checkout latency"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId":        "check-123",
				"includeHistory": true,
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), "include_history=true")
	})

	t.Run("null response body does not panic and emits empty details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`null`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"checkId": "check-123",
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
		require.Len(t, execCtx.Payloads, 1)

		emittedPayload, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, GetCheckDetailsPayloadType, emittedPayload["type"])
		require.IsType(t, time.Time{}, emittedPayload["timestamp"])

		data, ok := emittedPayload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "check-123", data["checkId"])
		assert.Equal(t, map[string]any{"checkId": "check-123"}, data["details"])
	})
}
