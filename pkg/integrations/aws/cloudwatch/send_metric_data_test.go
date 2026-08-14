package cloudwatch

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

func validSendMetricDataConfig() map[string]any {
	return map[string]any{
		"region":     "us-east-1",
		"namespace":  "MyApp/Orders",
		"metricName": "OrdersProcessed",
		"value":      42.0,
	}
}

func TestSendMetricData_Setup(t *testing.T) {
	component := &SendMetricData{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		config := validSendMetricDataConfig()
		config["region"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing namespace -> error", func(t *testing.T) {
		config := validSendMetricDataConfig()
		config["namespace"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("namespace starting with AWS/ -> error", func(t *testing.T) {
		config := validSendMetricDataConfig()
		config["namespace"] = "AWS/Custom"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "reserved for AWS services")
	})

	t.Run("missing metric name -> error", func(t *testing.T) {
		config := validSendMetricDataConfig()
		config["metricName"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "metric name is required")
	})

	t.Run("invalid timestamp -> error", func(t *testing.T) {
		config := validSendMetricDataConfig()
		config["timestamp"] = "not-a-date"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "invalid timestamp")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: validSendMetricDataConfig()})
		require.NoError(t, err)
	})
}

func TestSendMetricData_Execute(t *testing.T) {
	component := &SendMetricData{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  validSendMetricDataConfig(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "credentials")
	})

	t.Run("valid request -> publishes the metric", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`<PutMetricDataResponse></PutMetricDataResponse>`)),
				},
			},
		}

		config := validSendMetricDataConfig()
		config["unit"] = "Count"
		config["timestamp"] = "2026-05-29T09:00:00Z"
		config["dimensions"] = []map[string]any{{"key": "Environment", "value": "production"}}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		require.True(t, execState.Passed)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "MyApp/Orders", data["namespace"])
		assert.Equal(t, "OrdersProcessed", data["metricName"])
		assert.Equal(t, 42.0, data["value"])
		assert.Equal(t, "Count", data["unit"])
		assert.Equal(t, "2026-05-29T09:00:00Z", data["timestamp"])

		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		requestBody := string(body)
		assert.Contains(t, requestBody, "Action=PutMetricData")
		assert.Contains(t, requestBody, "Namespace=MyApp%2FOrders")
		assert.Contains(t, requestBody, "MetricData.member.1.MetricName=OrdersProcessed")
		assert.Contains(t, requestBody, "MetricData.member.1.Value=42")
		assert.Contains(t, requestBody, "MetricData.member.1.Dimensions.member.1.Name=Environment")
	})

	t.Run("datetime-local timestamp from the UI picker -> parsed as UTC", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`<PutMetricDataResponse></PutMetricDataResponse>`)),
				},
			},
		}

		config := validSendMetricDataConfig()
		config["timestamp"] = "2026-05-29T09:00"

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "2026-05-29T09:00:00Z", data["timestamp"])
	})

	t.Run("publish failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`<ErrorResponse><Error><Code>InvalidParameterValue</Code><Message>bad value</Message></Error></ErrorResponse>`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  validSendMetricDataConfig(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.ErrorContains(t, err, "failed to send metric data")
	})
}
