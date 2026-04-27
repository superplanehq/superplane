package grafana

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

func Test__CreateHTTPSyntheticCheck__Setup__ValidatesSpec(t *testing.T) {
	component := &CreateHTTPSyntheticCheck{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"job": "API health",
			"request": map[string]any{
				"target": "https://api.example.com/health",
				"method": "GET",
			},
			"schedule": map[string]any{
				"probes":    []string{},
				"timeout":   3000,
				"frequency": 60,
			},
		},
	})

	require.ErrorContains(t, err, "at least one probe is required")
}

func Test__CreateHTTPSyntheticCheck__Execute(t *testing.T) {
	component := &CreateHTTPSyntheticCheck{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			grafanaSyntheticHTTPResponse(`{
				"id": 101,
				"job": "API health",
				"target": "https://api.example.com/health",
				"frequency": 60000,
				"timeout": 3000,
				"enabled": true,
				"basicMetricsOnly": true,
				"settings": {"http": {"method": "GET"}},
				"probes": [1]
			}`),
			grafanaSyntheticHTTPResponse(`{}`),
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"job": "API health",
			"request": map[string]any{
				"target": "https://api.example.com/health",
				"method": "GET",
			},
			"schedule": map[string]any{
				"probes":    []string{"1"},
				"timeout":   3000,
				"frequency": 60,
			},
			"failIfHeaderMatchesRegexp": []map[string]any{
				{
					"header":       "X-Canary",
					"regexp":       "failed",
					"allowMissing": true,
				},
			},
			"alerts": []map[string]any{
				{
					"name":      "HTTPRequestDurationTooHighAvg",
					"threshold": 500,
					"period":    "5m",
				},
			},
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "grafana-token",
			},
		},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "grafana.syntheticCheck.created", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	payload := execCtx.Payloads[0].(map[string]any)
	data := payload["data"].(map[string]any)
	assert.Equal(t, "https://grafana.example.com/a/grafana-synthetic-monitoring-app/checks/101", data["checkUrl"])

	require.Len(t, httpContext.Requests, 3)
	body, err := io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)

	var requestPayload map[string]any
	require.NoError(t, json.Unmarshal(body, &requestPayload))
	assert.Equal(t, float64(60000), requestPayload["frequency"])
	settings := requestPayload["settings"].(map[string]any)["http"].(map[string]any)
	matches := settings["failIfHeaderMatchesRegexp"].([]any)
	require.Len(t, matches, 1)
	match := matches[0].(map[string]any)
	assert.Equal(t, "X-Canary", match["header"])
	assert.Equal(t, "failed", match["regexp"])
	assert.Equal(t, true, match["allowMissing"])
	assert.NotContains(t, settings, "tlsConfig")

	alertsBody, err := io.ReadAll(httpContext.Requests[2].Body)
	require.NoError(t, err)
	var alertPayload map[string]any
	require.NoError(t, json.Unmarshal(alertsBody, &alertPayload))
	alerts := alertPayload["alerts"].([]any)
	require.Len(t, alerts, 1)
	assert.Equal(t, "HTTPRequestDurationTooHighAvg", alerts[0].(map[string]any)["name"])
}

func Test__CreateHTTPSyntheticCheck__Execute__DeletesCheckWhenAlertConfigurationFails(t *testing.T) {
	component := &CreateHTTPSyntheticCheck{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			grafanaSyntheticHTTPResponse(`{
				"id": 101,
				"tenantId": 1,
				"job": "API health",
				"target": "https://api.example.com/health",
				"frequency": 60000,
				"timeout": 3000,
				"enabled": true,
				"basicMetricsOnly": true,
				"settings": {"http": {"method": "GET"}},
				"probes": [1]
			}`),
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`alert update failed`)),
			},
			grafanaSyntheticHTTPResponse(`{"deleted":true}`),
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"job": "API health",
			"request": map[string]any{
				"target": "https://api.example.com/health",
				"method": "GET",
			},
			"schedule": map[string]any{
				"probes":    []string{"1"},
				"timeout":   3000,
				"frequency": 60,
			},
			"alerts": []map[string]any{
				{
					"name":      "HTTPRequestDurationTooHighAvg",
					"threshold": 500,
					"period":    "5m",
				},
			},
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "grafana-token",
			},
		},
		ExecutionState: execCtx,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "error configuring synthetic check alerts")
	require.Empty(t, execCtx.Payloads)
	require.Len(t, httpContext.Requests, 4)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[3].Method)
	assert.Equal(t, "/api/datasources/proxy/uid/sm-ds/sm/check/delete/101", httpContext.Requests[3].URL.Path)
}
