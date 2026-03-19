package digitalocean

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

func Test__GetDropletMetrics__Setup(t *testing.T) {
	component := &GetDropletMetrics{}

	t.Run("missing droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"lookbackPeriod": "1h",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "droplet is required")
	})

	t.Run("missing lookbackPeriod returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet": "98765432",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"droplet": {"id": 98765432, "name": "test-droplet"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "lookbackPeriod is required")
	})

	t.Run("invalid lookbackPeriod returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet":        "98765432",
				"lookbackPeriod": "2h",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"droplet": {"id": 98765432, "name": "test-droplet"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "invalid lookbackPeriod")
	})

	t.Run("expression droplet is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet":        "{{ $.trigger.data.dropletId }}",
				"lookbackPeriod": "1h",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet":        "98765432",
				"lookbackPeriod": "24h",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"droplet": {"id": 98765432, "name": "test-droplet"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func metricsResponseWithValues(values string) string {
	return `{
		"status": "success",
		"data": {
			"resultType": "matrix",
			"result": [
				{
					"metric": {},
					"values": ` + values + `
				}
			]
		}
	}`
}

func Test__GetDropletMetrics__Execute(t *testing.T) {
	component := &GetDropletMetrics{}

	t.Run("successful fetch -> emits averaged metrics payload", func(t *testing.T) {
		// CPU: cumulative counter values per mode (idle, user, system, etc.).
		// idle delta = 3675-3600 = 75, user delta = 220-200 = 20, system delta = 105-100 = 5
		// total delta = 100, CPU usage = (100-75)/100 * 100 = 25%
		cpuResp := `{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [
					{"metric": {"mode": "idle"}, "values": [[1742205600, "3600.0"], [1742206200, "3675.0"]]},
					{"metric": {"mode": "user"}, "values": [[1742205600, "200.0"], [1742206200, "220.0"]]},
					{"metric": {"mode": "system"}, "values": [[1742205600, "100.0"], [1742206200, "105.0"]]},
					{"metric": {"mode": "iowait"}, "values": [[1742205600, "0.0"], [1742206200, "0.0"]]},
					{"metric": {"mode": "irq"}, "values": [[1742205600, "0.0"], [1742206200, "0.0"]]},
					{"metric": {"mode": "nice"}, "values": [[1742205600, "0.0"], [1742206200, "0.0"]]},
					{"metric": {"mode": "softirq"}, "values": [[1742205600, "0.0"], [1742206200, "0.0"]]},
					{"metric": {"mode": "steal"}, "values": [[1742205600, "0.0"], [1742206200, "0.0"]]}
				]
			}
		}`
		// Memory available: avg of 530000000 and 530000000 = 530000000
		memAvailableResp := metricsResponseWithValues(`[[1742205600, "530000000"], [1742206200, "530000000"]]`)
		// Memory total: avg of 1000000000 and 1000000000 = 1000000000
		memTotalResp := metricsResponseWithValues(`[[1742205600, "1000000000"], [1742206200, "1000000000"]]`)
		// Outbound bandwidth: DO API returns gauge values already in Mbps.
		// avg of 1.2 and 0.8 = 1.0 Mbps
		outboundResp := metricsResponseWithValues(`[[1742205600, "1.2"], [1742206200, "0.8"]]`)
		// Inbound bandwidth: DO API returns gauge values already in Mbps.
		// avg of 0.6 and 0.4 = 0.5 Mbps
		inboundResp := metricsResponseWithValues(`[[1742205600, "0.6"], [1742206200, "0.4"]]`)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(cpuResp))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(memAvailableResp))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(memTotalResp))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(outboundResp))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(inboundResp))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"droplet":        "98765432",
				"lookbackPeriod": "1h",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.droplet.metrics", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "98765432", payload["dropletId"])
		assert.Equal(t, "1h", payload["lookbackPeriod"])
		assert.NotEmpty(t, payload["start"])
		assert.NotEmpty(t, payload["end"])
		assert.Equal(t, 25.0, payload["avgCpuUsagePercent"])
		assert.Equal(t, 47.0, payload["avgMemoryUsagePercent"])
		assert.Equal(t, 1.0, payload["avgPublicOutboundBandwidthMbps"])
		assert.Equal(t, 0.5, payload["avgPublicInboundBandwidthMbps"])
	})

	t.Run("invalid lookbackPeriod -> returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"droplet":        "98765432",
				"lookbackPeriod": "2h",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid lookbackPeriod")
		assert.False(t, executionState.Passed)
	})

	t.Run("CPU metrics API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"Unable to authenticate you."}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "bad-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"droplet":        "98765432",
				"lookbackPeriod": "1h",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get CPU metrics")
		assert.False(t, executionState.Passed)
	})
}
