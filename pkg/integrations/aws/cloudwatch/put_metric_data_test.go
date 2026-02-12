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

func Test__PutMetricData__Setup(t *testing.T) {
	component := &PutMetricData{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    " ",
				"namespace": "MyService/Production",
				"metricData": []any{
					map[string]any{
						"metricName": "RequestCount",
						"value":      1,
					},
				},
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing namespace -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": " ",
				"metricData": []any{
					map[string]any{
						"metricName": "RequestCount",
						"value":      1,
					},
				},
			},
		})

		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("missing metric data -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "MyService/Production",
			},
		})

		require.ErrorContains(t, err, "metric data is required")
	})

	t.Run("missing metric value -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "MyService/Production",
				"metricData": []any{
					map[string]any{
						"metricName": "RequestCount",
					},
				},
			},
		})

		require.ErrorContains(t, err, "metricData[0].value is required")
	})

	t.Run("invalid storage resolution -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "MyService/Production",
				"metricData": []any{
					map[string]any{
						"metricName":        "RequestCount",
						"value":             1,
						"storageResolution": "5",
					},
				},
			},
		})

		require.ErrorContains(t, err, "metricData[0].storageResolution must be 1 or 60")
	})

	t.Run("invalid dimension -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "MyService/Production",
				"metricData": []any{
					map[string]any{
						"metricName": "RequestCount",
						"value":      1,
						"dimensions": []any{
							map[string]any{"name": "Service", "value": " "},
						},
					},
				},
			},
		})

		require.ErrorContains(t, err, "metricData[0].dimensions[0].value is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "MyService/Production",
				"metricData": []any{
					map[string]any{
						"metricName":        "RequestCount",
						"value":             42.0,
						"unit":              "Count",
						"timestamp":         "2026-02-10T12:00",
						"storageResolution": "1",
						"dimensions": []any{
							map[string]any{"name": "Service", "value": "api"},
						},
					},
				},
			},
		})

		require.NoError(t, err)
	})
}

func Test__PutMetricData__Execute(t *testing.T) {
	component := &PutMetricData{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "MyService/Production",
				"metricData": []any{
					map[string]any{
						"metricName": "RequestCount",
						"value":      1,
					},
				},
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<PutMetricDataResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
							<ResponseMetadata>
								<RequestId>req-123</RequestId>
							</ResponseMetadata>
						</PutMetricDataResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"namespace": "MyService/Production",
				"metricData": []any{
					map[string]any{
						"metricName":        "RequestCount",
						"value":             42.0,
						"unit":              "Count",
						"timestamp":         "2026-02-10T12:00",
						"storageResolution": "1",
						"dimensions": []any{
							map[string]any{"name": "Service", "value": "api"},
						},
					},
					map[string]any{
						"metricName": "LatencyMs",
						"value":      18.5,
						"unit":       "Milliseconds",
					},
				},
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		payload := execState.Payloads[0].(map[string]any)["data"]
		output, ok := payload.(PutMetricDataOutput)
		require.True(t, ok)
		assert.Equal(t, "req-123", output.RequestID)
		assert.Equal(t, "us-east-1", output.Region)
		assert.Equal(t, "MyService/Production", output.Namespace)
		assert.Equal(t, 2, output.MetricCount)
		assert.Equal(t, []string{"RequestCount", "LatencyMs"}, output.MetricNames)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, "https://monitoring.us-east-1.amazonaws.com/", request.URL.String())

		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		bodyText := string(body)
		assert.Contains(t, bodyText, "Action=PutMetricData")
		assert.Contains(t, bodyText, "Namespace=MyService%2FProduction")
		assert.Contains(t, bodyText, "MetricData.member.1.MetricName=RequestCount")
		assert.Contains(t, bodyText, "MetricData.member.1.Value=42")
		assert.Contains(t, bodyText, "MetricData.member.1.StorageResolution=1")
		assert.Contains(t, bodyText, "MetricData.member.1.Dimensions.member.1.Name=Service")
		assert.Contains(t, bodyText, "MetricData.member.2.MetricName=LatencyMs")
	})
}
