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

func Test__UpdateSyntheticCheck__Setup(t *testing.T) {
	component := UpdateSyntheticCheck{}

	t.Run("origin is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"originOrId": "",
				"name":       "checkout-health",
				"enabled":    true,
				"pluginKind": "http",
				"method":     "get",
				"url":        "https://example.com",
			},
		})

		require.ErrorContains(t, err, "originOrId is required")
	})
}

func Test__UpdateSyntheticCheck__Execute(t *testing.T) {
	component := UpdateSyntheticCheck{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"updated"}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"originOrId": "checkout-health-check",
			"name":       "checkout-health",
			"enabled":    true,
			"pluginKind": "http",
			"method":     "get",
			"url":        "https://example.com/health",
			"headers": []map[string]any{
				{
					"key":   "Accept",
					"value": "application/json",
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
	assert.Equal(t, UpdateSyntheticCheckPayloadType, execCtx.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/synthetic-checks/checkout-health-check")
	body, readErr := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, readErr)

	assert.Contains(t, string(body), `"metadata":{"name":"checkout-health"}`)
	assert.Contains(t, string(body), `"method":"get"`)
	assert.Contains(t, string(body), `"url":"https://example.com/health"`)
	assert.Contains(t, string(body), `"headers":{"Accept":"application/json"}`)
}
