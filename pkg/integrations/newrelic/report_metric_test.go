package newrelic

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

func Test__ReportMetric__Setup(t *testing.T) {
	component := &ReportMetric{}

	t.Run("missing metricName -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "",
				"metricType": "gauge",
				"value":      1,
			},
		})

		require.ErrorContains(t, err, "metricName is required")
	})

	t.Run("missing metricType -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "custom.metric",
				"metricType": "",
				"value":      1,
			},
		})

		require.ErrorContains(t, err, "metricType is required")
	})

	t.Run("invalid metricType -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "custom.metric",
				"metricType": "invalid",
				"value":      1,
			},
		})

		require.ErrorContains(t, err, "invalid metricType")
	})

	t.Run("missing value -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "custom.metric",
				"metricType": "gauge",
			},
		})

		require.ErrorContains(t, err, "value is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"metricName": "custom.deployment.count",
				"metricType": "count",
				"value":      1,
			},
		})

		require.NoError(t, err)
	})
}

func Test__ReportMetric__Execute(t *testing.T) {
	component := &ReportMetric{}

	t.Run("successful metric report", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"requestId": "abc123"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"metricName": "custom.deployment.count",
				"metricType": "count",
				"value":      1,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "newrelic.metric", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "metric-api.newrelic.com/metric/v1")
		assert.Equal(t, "test-license-key", req.Header.Get("Api-Key"))
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Invalid payload"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"metricName": "custom.metric",
				"metricType": "gauge",
				"value":      1,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to report metric")
	})
}

func Test__ReportMetric__OutputChannels(t *testing.T) {
	component := &ReportMetric{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}
