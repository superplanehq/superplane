package grafana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetHTTPSyntheticCheck__Execute__EmptyChannelWhenNoOutcome(t *testing.T) {
	component := &GetHTTPSyntheticCheck{}
	responses := grafanaGetCheckResponses("1")
	responses[7] = grafanaSyntheticHTTPResponse(`{"results":{}}`)

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
	responses[3] = grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[144.20027816411684]]}}]}}}`)
	responses[4] = grafanaSyntheticHTTPResponse(`{"results":{"A":{"frames":[{"data":{"values":[[1],[144.20027816411684]]}}]}}}`)

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
