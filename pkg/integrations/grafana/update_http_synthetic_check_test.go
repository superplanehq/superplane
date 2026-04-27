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

func Test__UpdateHTTPSyntheticCheck__Setup__AllowsExpression(t *testing.T) {
	component := &UpdateHTTPSyntheticCheck{}
	httpContext := &contexts.HTTPContext{}
	metadata := &contexts.MetadataContext{}
	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"syntheticCheck": "{{ $['Create HTTP Synthetic Check'].data.check.id }}",
		},
		HTTP:     httpContext,
		Metadata: metadata,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})
	require.NoError(t, err)
	require.Empty(t, httpContext.Requests)
}

func Test__UpdateHTTPSyntheticCheck__Execute__RejectsUnresolvedExpression(t *testing.T) {
	component := &UpdateHTTPSyntheticCheck{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"syntheticCheck": "{{ $['x'].id }}",
		},
		Integration:    &contexts.IntegrationContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolve")
}

func Test__UpdateHTTPSyntheticCheck__Execute(t *testing.T) {
	component := &UpdateHTTPSyntheticCheck{}
	checkJSON := `{
		"id": 101,
		"tenantId": 1,
		"job": "API health",
		"target": "https://api.example.com/health",
		"frequency": 30000,
		"timeout": 5000,
		"enabled": true,
		"alertSensitivity": "medium",
		"basicMetricsOnly": false,
		"settings": {"http": {
			"method": "GET",
			"ipVersion": "V6",
			"compression": "gzip",
			"tlsConfig": {"serverName": "api.example.com", "insecureSkipVerify": true},
			"failIfHeaderNotMatchesRegexp": [{"header": "content-type", "regexp": "application/json", "allowMissing": true}]
		}},
		"probes": [1,2]
	}`
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			grafanaSyntheticCheckGetResponse(checkJSON),
			grafanaSyntheticHTTPResponse(checkJSON),
			grafanaSyntheticHTTPResponse(`{}`),
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"syntheticCheck": "101",
			"job":            "API health",
			"request": map[string]any{
				"target": "https://api.example.com/health",
				"method": "GET",
				"body":   "  {\"ping\": true}\n",
			},
			"schedule": map[string]any{
				"probes":    []string{"1", "2"},
				"timeout":   5000,
				"frequency": 30,
			},
			"alerts": []map[string]any{
				{
					"name":      "ProbeFailedExecutionsTooHigh",
					"threshold": 2,
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
	assert.Equal(t, "grafana.syntheticCheck.updated", execCtx.Type)
	require.Len(t, httpContext.Requests, 4)

	body, err := io.ReadAll(httpContext.Requests[2].Body)
	require.NoError(t, err)
	var requestPayload map[string]any
	require.NoError(t, json.Unmarshal(body, &requestPayload))
	assert.Equal(t, "medium", requestPayload["alertSensitivity"])
	assert.Equal(t, false, requestPayload["basicMetricsOnly"])
	settings := requestPayload["settings"].(map[string]any)["http"].(map[string]any)
	assert.Equal(t, "  {\"ping\": true}\n", settings["body"])
	assert.Equal(t, "V6", settings["ipVersion"])
	assert.Equal(t, "gzip", settings["compression"])
	tlsConfig := settings["tlsConfig"].(map[string]any)
	assert.Equal(t, "api.example.com", tlsConfig["serverName"])
	assert.Equal(t, true, tlsConfig["insecureSkipVerify"])
	headerNotMatches := settings["failIfHeaderNotMatchesRegexp"].([]any)
	require.Len(t, headerNotMatches, 1)
	headerNotMatch := headerNotMatches[0].(map[string]any)
	assert.Equal(t, "content-type", headerNotMatch["header"])
	assert.Equal(t, "application/json", headerNotMatch["regexp"])
	assert.Equal(t, true, headerNotMatch["allowMissing"])
}

func Test__UpdateHTTPSyntheticCheck__Execute__PreservesExistingAlertsWhenAlertsSectionOmitted(t *testing.T) {
	component := &UpdateHTTPSyntheticCheck{}
	checkJSON := `{
		"id": 101,
		"tenantId": 1,
		"job": "API health",
		"target": "https://api.example.com/health",
		"frequency": 30000,
		"timeout": 5000,
		"enabled": true,
		"basicMetricsOnly": true,
		"settings": {"http": {"method": "GET"}},
		"probes": [1,2]
	}`
	alertsJSON := `{"alerts":[{"name":"ProbeFailedExecutionsTooHigh","threshold":2,"period":"5m"}]}`
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			grafanaSyntheticCheckGetResponse(checkJSON),
			grafanaSyntheticHTTPResponse(checkJSON),
			grafanaSyntheticHTTPResponse(alertsJSON),
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"syntheticCheck": "101",
			"job":            "API health",
			"request": map[string]any{
				"target": "https://api.example.com/health",
				"method": "GET",
			},
			"schedule": map[string]any{
				"probes":    []string{"1", "2"},
				"timeout":   5000,
				"frequency": 30,
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
	require.Len(t, httpContext.Requests, 4)
	assert.Equal(t, http.MethodGet, httpContext.Requests[3].Method)
	assert.True(t, strings.HasSuffix(httpContext.Requests[3].URL.Path, "/sm/check/101/alerts"))

	require.Len(t, execCtx.Payloads, 1)
	payload := execCtx.Payloads[0].(map[string]any)
	data := payload["data"].(map[string]any)
	alerts := data["alerts"].([]SyntheticCheckAlert)
	require.Len(t, alerts, 1)
	assert.Equal(t, "ProbeFailedExecutionsTooHigh", alerts[0].Name)

	check := data["check"].(*SyntheticCheck)
	require.Len(t, check.Alerts, 1)
	assert.Equal(t, int64(2), check.Alerts[0].Threshold)
}
