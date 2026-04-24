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

func TestNormalizeSyntheticFrequency_treatsInputAsSeconds(t *testing.T) {
	assert.Equal(t, int64(1000000), normalizeSyntheticFrequency(1000))
	assert.Equal(t, int64(2000000), normalizeSyntheticFrequency(2000))
}

func TestSyntheticCheckToSpecBase_preservesExactGrafanaFrequency(t *testing.T) {
	base, err := syntheticCheckToSpecBase(&SyntheticCheck{
		Job:       "API health",
		Target:    "https://api.example.com/health",
		Frequency: 1500,
		Timeout:   3000,
		Enabled:   true,
		Probes:    []int64{1},
		Settings: SyntheticCheckSettings{
			HTTP: &SyntheticCheckHTTPSettings{Method: "GET"},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, base.FrequencyMilliseconds)
	assert.Equal(t, int64(2), base.Frequency)
	assert.Equal(t, int64(1500), *base.FrequencyMilliseconds)

	payload, err := buildSyntheticCheckPayload(base)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), payload.Frequency)

	base.Frequency = 1000
	base.FrequencyMilliseconds = nil
	payload, err = buildSyntheticCheckPayload(base)
	require.NoError(t, err)
	assert.Equal(t, int64(1000000), payload.Frequency)
}

func grafanaSyntheticDataSourceResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`[{"uid":"sm-ds","name":"Synthetic Monitoring","type":"synthetic-monitoring-datasource","jsonData":{"metrics":{"uid":"prom-ds"}}}]`)),
	}
}

func grafanaSyntheticCheckListResponse(checkJSON string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("[" + checkJSON + "]")),
	}
}

func grafanaSyntheticCheckGetResponse(checkJSON string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(checkJSON)),
	}
}

func Test__CreateHTTPSyntheticCheck__Execute(t *testing.T) {
	component := &CreateHTTPSyntheticCheck{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id": 101,
					"job": "API health",
					"target": "https://api.example.com/health",
					"frequency": 60000,
					"timeout": 3000,
					"enabled": true,
					"basicMetricsOnly": true,
					"settings": {"http": {"method": "GET"}},
					"probes": [1]
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
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

func grafanaGetCheckResponses(reachability string) []*http.Response {
	checkJSON := `{
		"id": 101,
		"job": "API health",
		"target": "https://api.example.com/health",
		"frequency": 60000,
		"timeout": 3000,
		"enabled": true,
		"basicMetricsOnly": true,
		"settings": {"http": {"method": "GET"}},
		"probes": [1]
	}`

	return []*http.Response{
		grafanaSyntheticDataSourceResponse(),
		grafanaSyntheticCheckGetResponse(checkJSON),
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"alerts":[{"name":"ProbeFailedExecutionsTooHigh","threshold":1,"period":"5m"}]}`)),
		},
		// success runs
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1438]]}}]}}}`)),
		},
		// total runs
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1440]]}}]}}}`)),
		},
		// avg latency
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[0.142]]}}]}}}`)),
		},
		// uptime
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[0.999]]}}]}}}`)),
		},
		// reachability
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[` + reachability + `]]}}]}}}`)),
		},
		// last execution
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1713176700]]}}]}}}`)),
		},
		// ssl earliest expiry
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[1715768700]]}}]}}}`)),
		},
	}
}

func Test__GetHTTPSyntheticCheck__Execute__EmptyChannelWhenNoOutcome(t *testing.T) {
	component := &GetHTTPSyntheticCheck{}
	responses := grafanaGetCheckResponses("1")
	// Outcome query (probe_success avg); empty result → no LastOutcome → empty channel
	responses[7] = &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"results":{}}`)),
	}

	httpContext := &contexts.HTTPContext{Responses: responses}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"syntheticCheck": "101",
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
	assert.Equal(t, "", execCtx.Channel)
}

func Test__GetHTTPSyntheticCheck__Execute__ReturnsConfigurationAndMetrics(t *testing.T) {
	component := &GetHTTPSyntheticCheck{}

	tests := []struct {
		name         string
		reachability string
		wantChannel  string
	}{
		{name: "up when all probe locations pass", reachability: "1", wantChannel: "up"},
		{name: "partial when some probe locations pass and some fail", reachability: "0.5", wantChannel: "partial"},
		{name: "down when all probe locations fail", reachability: "0", wantChannel: "down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpContext := &contexts.HTTPContext{Responses: grafanaGetCheckResponses(tt.reachability)}
			execCtx := &contexts.ExecutionStateContext{}
			err := component.Execute(core.ExecutionContext{
				Configuration: map[string]any{
					"syntheticCheck": "101",
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
			assert.Equal(t, "grafana.syntheticCheck", execCtx.Type)
			assert.Equal(t, tt.wantChannel, execCtx.Channel)
			require.Len(t, execCtx.Payloads, 1)
			payload := execCtx.Payloads[0].(map[string]any)
			data := payload["data"].(map[string]any)
			assert.Contains(t, data, "configuration")
			assert.Contains(t, data, "metrics")
			metrics := data["metrics"].(*SyntheticCheckMetrics)
			require.NotNil(t, metrics.ReachabilityPercent24h)
			require.NotNil(t, metrics.UptimePercent24h)
			require.NotNil(t, metrics.SSLEarliestExpiryAt)
			require.NotNil(t, metrics.FrequencyMilliseconds)
			assert.Equal(t, float64(60000), float64(*metrics.FrequencyMilliseconds))
		})
	}
}

func Test__GetHTTPSyntheticCheck__Execute__RoundsFractionalRunCounts(t *testing.T) {
	component := &GetHTTPSyntheticCheck{}
	responses := grafanaGetCheckResponses("1")
	responses[3] = &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[144.20027816411684]]}}]}}}`)),
	}
	responses[4] = &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"results":{"A":{"frames":[{"data":{"values":[[1],[144.20027816411684]]}}]}}}`)),
	}

	httpContext := &contexts.HTTPContext{Responses: responses}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"syntheticCheck": "101",
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
	require.Len(t, execCtx.Payloads, 1)

	payload := execCtx.Payloads[0].(map[string]any)
	data := payload["data"].(map[string]any)
	metrics := data["metrics"].(*SyntheticCheckMetrics)

	require.NotNil(t, metrics.SuccessRuns24h)
	require.NotNil(t, metrics.FailureRuns24h)
	require.NotNil(t, metrics.TotalRuns24h)
	assert.Equal(t, 144.0, *metrics.SuccessRuns24h)
	assert.Equal(t, 0.0, *metrics.FailureRuns24h)
	assert.Equal(t, 144.0, *metrics.TotalRuns24h)
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
					"basicMetricsOnly": true,
					"settings": {"http": {"method": "GET", "tlsConfig": {"serverName": "api.example.com", "insecureSkipVerify": true}}},
					"probes": [1,2]
				}`
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			grafanaSyntheticCheckGetResponse(checkJSON),
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(checkJSON)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
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
	settings := requestPayload["settings"].(map[string]any)["http"].(map[string]any)
	tlsConfig := settings["tlsConfig"].(map[string]any)
	assert.Equal(t, "api.example.com", tlsConfig["serverName"])
	assert.Equal(t, true, tlsConfig["insecureSkipVerify"])
}

func Test__DeleteHTTPSyntheticCheck__Execute(t *testing.T) {
	component := &DeleteHTTPSyntheticCheck{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			grafanaSyntheticCheckGetResponse(`{
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
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"msg":"Check deleted","checkId":101}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"syntheticCheck": "101",
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
	assert.Equal(t, "grafana.syntheticCheck.deleted", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	payload := execCtx.Payloads[0].(map[string]any)
	data := payload["data"].(DeleteHTTPSyntheticCheckOutput)
	assert.Equal(t, "101", data.SyntheticCheck)
	assert.True(t, data.Deleted)
}
