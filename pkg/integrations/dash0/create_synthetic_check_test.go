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

func Test__CreateSyntheticCheck__Setup(t *testing.T) {
	component := CreateSyntheticCheck{}

	t.Run("name is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"method": "get",
				"url":    "https://example.com/health",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("legacy spec remains supported", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `[{"kind":"Dash0SyntheticCheck","metadata":{"name":"checkout-health"},"spec":{"enabled":true,"plugin":{"kind":"http","spec":{"request":{"method":"get","url":"https://example.com"}}}}}]`,
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateSyntheticCheck__Execute(t *testing.T) {
	component := CreateSyntheticCheck{}

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
			"name":       "checkout-health",
			"enabled":    true,
			"pluginKind": "http",
			"method":     "get",
			"url":        "https://example.com/health",
			"headers": []map[string]any{
				{"key": "x-test", "value": "superplane"},
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
	assert.Equal(t, CreateSyntheticCheckPayloadType, execCtx.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/synthetic-checks/superplane-synthetic-")
	assert.Equal(t, "default", httpContext.Requests[0].URL.Query().Get("dataset"))

	requestBody, readErr := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, readErr)
	assert.Contains(t, string(requestBody), `"kind":"Dash0SyntheticCheck"`)
	assert.Contains(t, string(requestBody), `"name":"checkout-health"`)
	assert.Contains(t, string(requestBody), `"method":"get"`)
	assert.Contains(t, string(requestBody), `"url":"https://example.com/health"`)
}
