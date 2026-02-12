package dash0

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateCheckRule__Setup(t *testing.T) {
	component := CreateCheckRule{}

	t.Run("missing name fails setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"expression": "vector(1)",
			},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("legacy spec remains supported", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `{"groups":[{"name":"checkout.rules","rules":[{"alert":"CheckoutErrors","expr":"vector(1)"}]}]}`,
			},
		})
		require.NoError(t, err)
	})
}

func Test__CreateCheckRule__Execute(t *testing.T) {
	component := CreateCheckRule{}

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
			"originOrId":    "checkout-errors",
			"name":          "CheckoutErrors",
			"expression":    `sum(rate(http_requests_total{service="checkout",status=~"5.."}[5m])) > 0`,
			"for":           "5m",
			"interval":      "1m",
			"keepFiringFor": "10m",
			"labels": []map[string]any{
				{"key": "severity", "value": "warning"},
			},
			"annotations": []map[string]any{
				{"key": "summary", "value": "Checkout errors are above baseline"},
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
	assert.Equal(t, CreateCheckRulePayloadType, execCtx.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/alerting/check-rules/checkout-errors")
	assert.Equal(t, "default", httpContext.Requests[0].URL.Query().Get("dataset"))

	requestBody, readErr := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, readErr)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(requestBody, &payload))
	assert.Equal(t, "CheckoutErrors", payload["name"])
	assert.Equal(t, "sum(rate(http_requests_total{service=\"checkout\",status=~\"5..\"}[5m])) > 0", payload["expression"])
	assert.Equal(t, "1m", payload["interval"])
	assert.Equal(t, "5m", payload["for"])
	assert.Equal(t, "10m", payload["keepFiringFor"])
	assert.Equal(t, map[string]any{"severity": "warning"}, payload["labels"])
	assert.Equal(t, map[string]any{"summary": "Checkout errors are above baseline"}, payload["annotations"])
}
