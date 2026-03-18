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

func metricsResponse(metricType string) string {
	return `{
		"status": "success",
		"data": {
			"resultType": "matrix",
			"result": [
				{
					"metric": {},
					"values": [
						[1742205600, "12.4"],
						[1742206200, "15.1"]
					]
				}
			]
		}
	}`
}

func Test__GetDropletMetrics__Execute(t *testing.T) {
	component := &GetDropletMetrics{}

	t.Run("successful fetch -> emits combined metrics payload", func(t *testing.T) {
		mr := metricsResponse("cpu")
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// CPU metrics
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(mr))},
				// Memory metrics
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(mr))},
				// Public outbound bandwidth
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(mr))},
				// Public inbound bandwidth
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(mr))},
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
		assert.NotNil(t, payload["cpu"])
		assert.NotNil(t, payload["memory"])
		assert.NotNil(t, payload["publicOutboundBandwidth"])
		assert.NotNil(t, payload["publicInboundBandwidth"])
		assert.NotEmpty(t, payload["start"])
		assert.NotEmpty(t, payload["end"])
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
