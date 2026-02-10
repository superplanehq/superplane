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
			"spec": `{"kind":"Dash0SyntheticCheck","metadata":{"name":"checkout-health"},"spec":{"enabled":true,"plugin":{"kind":"http","spec":{"request":{"method":"get","url":"https://example.com"}}}}}`,
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
			"spec":       `{"kind":"Dash0SyntheticCheck","metadata":{"name":"checkout-health"},"spec":{"enabled":true,"plugin":{"kind":"http","spec":{"request":{"method":"get","url":"https://example.com/health"}}}}}`,
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
	assert.Equal(t, "default", httpContext.Requests[0].URL.Query().Get("dataset"))
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
			"originOrId": "checkout-errors",
			"spec":       `{"groups":[{"name":"checkout.rules","interval":"1m","rules":[{"alert":"CheckoutErrors","expr":"sum(rate(http_requests_total{service=\"checkout\",status=~\"5..\"}[5m])) > 0","for":"5m","labels":{"severity":"warning"},"annotations":{"summary":"Checkout errors are above baseline"}}]}]}`,
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
}

func Test__UpdateCheckRule__Execute(t *testing.T) {
	component := UpdateCheckRule{}

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
			"originOrId": "checkout-errors",
			"spec":       `{"name":"CheckoutErrors","expression":"sum(rate(http_requests_total{service=\"checkout\",status=~\"5..\"}[5m])) > 1","for":"10m","labels":{"severity":"critical"},"annotations":{"summary":"Checkout errors are critical"}}`,
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
	assert.Equal(t, UpdateCheckRulePayloadType, execCtx.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/alerting/check-rules/checkout-errors")
	assert.Equal(t, "default", httpContext.Requests[0].URL.Query().Get("dataset"))

	requestBody, readErr := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, readErr)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(requestBody, &payload))
	assert.Equal(t, "CheckoutErrors", payload["name"])
	assert.Equal(t, "sum(rate(http_requests_total{service=\"checkout\",status=~\"5..\"}[5m])) > 1", payload["expression"])
	assert.Equal(t, "10m", payload["for"])
}

func Test__UpsertComponents__SetupValidation(t *testing.T) {
	createSynthetic := CreateSyntheticCheck{}
	updateSynthetic := UpdateSyntheticCheck{}
	createRule := CreateCheckRule{}
	updateRule := UpdateCheckRule{}

	t.Run("invalid synthetic spec fails setup", func(t *testing.T) {
		err := createSynthetic.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": "not-json",
			},
		})
		require.ErrorContains(t, err, "parse spec as JSON object")
	})

	t.Run("single-item array synthetic spec passes setup", func(t *testing.T) {
		err := createSynthetic.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `[{"kind":"Dash0SyntheticCheck","metadata":{"name":"checkout-health"},"spec":{"enabled":true,"plugin":{"kind":"http","spec":{"request":{"method":"get","url":"https://example.com"}}}}}]`,
			},
		})
		require.NoError(t, err)
	})

	t.Run("missing synthetic kind fails setup", func(t *testing.T) {
		err := createSynthetic.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `{"spec":{"enabled":true,"plugin":{"kind":"http"}}}`,
			},
		})
		require.ErrorContains(t, err, "spec.kind is required")
	})

	t.Run("missing synthetic plugin kind fails setup", func(t *testing.T) {
		err := createSynthetic.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `{"kind":"Dash0SyntheticCheck","spec":{"enabled":true,"plugin":{"spec":{"request":{"method":"get","url":"https://example.com"}}}}}`,
			},
		})
		require.ErrorContains(t, err, "spec.spec.plugin.kind is required")
	})

	t.Run("update synthetic requires origin", func(t *testing.T) {
		err := updateSynthetic.Setup(core.SetupContext{
			Configuration: map[string]any{
				"originOrId": "",
				"spec":       `{"kind":"Dash0SyntheticCheck","metadata":{"name":"checkout-health"},"spec":{"enabled":true,"plugin":{"kind":"http","spec":{"request":{"method":"get","url":"https://example.com"}}}}}`,
			},
		})
		require.ErrorContains(t, err, "originOrId is required")
	})

	t.Run("invalid check rule spec fails setup", func(t *testing.T) {
		err := createRule.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": "{",
			},
		})
		require.ErrorContains(t, err, "parse spec as JSON object")
	})

	t.Run("multi-item array check rule spec fails setup", func(t *testing.T) {
		err := createRule.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `[{"groups":[{"name":"one","rules":[]} ]},{"groups":[{"name":"two","rules":[]}]}]`,
			},
		})
		require.ErrorContains(t, err, "must be a JSON object or a single-item JSON array")
	})

	t.Run("check rule spec requires expression", func(t *testing.T) {
		err := createRule.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `{"name":"CheckoutErrors"}`,
			},
		})
		require.ErrorContains(t, err, "spec.expression is required")
	})

	t.Run("prometheus style check rule supports single alert rule", func(t *testing.T) {
		err := createRule.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `{"groups":[{"name":"checkout.rules","rules":[{"alert":"CheckoutErrors","expr":"vector(1)"}]}]}`,
			},
		})
		require.NoError(t, err)
	})

	t.Run("prometheus style check rule rejects multiple alert rules", func(t *testing.T) {
		err := createRule.Setup(core.SetupContext{
			Configuration: map[string]any{
				"spec": `{"groups":[{"name":"checkout.rules","rules":[{"alert":"CheckoutErrors","expr":"vector(1)"},{"alert":"CheckoutLatency","expr":"vector(2)"}]}]}`,
			},
		})
		require.ErrorContains(t, err, "must contain exactly one alert rule")
	})

	t.Run("update check rule requires origin", func(t *testing.T) {
		err := updateRule.Setup(core.SetupContext{
			Configuration: map[string]any{
				"originOrId": "",
				"spec":       `{"groups":[]}`,
			},
		})
		require.ErrorContains(t, err, "originOrId is required")
	})
}
