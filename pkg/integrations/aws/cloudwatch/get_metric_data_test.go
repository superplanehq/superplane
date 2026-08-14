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

func encodedTestDimensions(t *testing.T, dimensions ...Dimension) string {
	t.Helper()
	encoded, err := EncodeDimensions(dimensions)
	require.NoError(t, err)
	return encoded
}

func validGetMetricDataConfig(t *testing.T) map[string]any {
	return map[string]any{
		"region":         "us-east-1",
		"service":        metricDataServiceECS,
		"dimensionsEcs":  encodedTestDimensions(t, Dimension{Name: "ClusterName", Value: "prod"}, Dimension{Name: "ServiceName", Value: "checkout-api"}),
		"lookbackPeriod": "1h",
	}
}

func TestGetMetricData_Setup(t *testing.T) {
	component := &GetMetricData{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		config := validGetMetricDataConfig(t)
		config["region"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("invalid service -> error", func(t *testing.T) {
		config := validGetMetricDataConfig(t)
		config["service"] = "lambda"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "invalid service")
	})

	t.Run("ecs missing dimensions -> error", func(t *testing.T) {
		config := validGetMetricDataConfig(t)
		delete(config, "dimensionsEcs")
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "ECS service is required")
	})

	t.Run("rds missing metric -> error", func(t *testing.T) {
		config := map[string]any{
			"region":         "us-east-1",
			"service":        metricDataServiceRDS,
			"dimensionsRds":  encodedTestDimensions(t, Dimension{Name: "DBInstanceIdentifier", Value: "prod-db"}),
			"lookbackPeriod": "1h",
		}
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "metric name is required")
	})

	t.Run("rds missing dimensions -> error", func(t *testing.T) {
		config := map[string]any{
			"region":         "us-east-1",
			"service":        metricDataServiceRDS,
			"rdsMetric":      metricNameRDSDatabaseConnections,
			"lookbackPeriod": "1h",
		}
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "RDS instance is required")
	})

	t.Run("valid ecs configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: validGetMetricDataConfig(t)})
		require.NoError(t, err)
	})

	t.Run("valid sqs configuration -> no error", func(t *testing.T) {
		config := map[string]any{
			"region":         "us-east-1",
			"service":        metricDataServiceSQS,
			"sqsMetric":      metricNameSQSMessagesVisible,
			"dimensionsSqs":  encodedTestDimensions(t, Dimension{Name: "QueueName", Value: "orders"}),
			"lookbackPeriod": "1h",
		}
		err := component.Setup(core.SetupContext{Configuration: config})
		require.NoError(t, err)
	})
}

func TestGetMetricData_Execute(t *testing.T) {
	component := &GetMetricData{}

	t.Run("invalid configuration -> fails execution", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.False(t, execState.Passed)
		assert.Contains(t, execState.FailureMessage, "failed to decode configuration")
	})

	t.Run("missing credentials -> fails execution", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validGetMetricDataConfig(t),
			ExecutionState: execState,
			Integration:    &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.False(t, execState.Passed)
		assert.Contains(t, execState.FailureMessage, "credentials")
	})

	t.Run("valid ecs request -> emits datapoints and aggregated average", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetMetricStatisticsResponse>
						  <GetMetricStatisticsResult>
						    <Datapoints>
						      <member><Timestamp>2026-05-29T08:05:00Z</Timestamp><Average>38.42</Average></member>
						      <member><Timestamp>2026-05-29T08:10:00Z</Timestamp><Average>45.91</Average></member>
						    </Datapoints>
						  </GetMetricStatisticsResult>
						</GetMetricStatisticsResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validGetMetricDataConfig(t),
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		require.True(t, execState.Passed)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "AWS/ECS", data["namespace"])
		assert.Equal(t, "CPUUtilization", data["metricName"])
		assert.Equal(t, "Average", data["statistic"])
		assert.Equal(t, 42.17, data["aggregatedValue"])

		datapoints := data["datapoints"].([]map[string]any)
		require.Len(t, datapoints, 2)
		assert.Equal(t, 38.42, datapoints[0]["value"])

		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Action=GetMetricStatistics")
		assert.Contains(t, string(body), "Dimensions.member.1.Name=ClusterName")
	})

	t.Run("alb request uses Sum statistic and totals datapoints", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetMetricStatisticsResponse>
						  <GetMetricStatisticsResult>
						    <Datapoints>
						      <member><Timestamp>2026-05-29T08:05:00Z</Timestamp><Sum>10</Sum></member>
						      <member><Timestamp>2026-05-29T08:10:00Z</Timestamp><Sum>15</Sum></member>
						    </Datapoints>
						  </GetMetricStatisticsResult>
						</GetMetricStatisticsResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		config := map[string]any{
			"region":         "us-east-1",
			"service":        metricDataServiceALB,
			"dimensionsAlb":  encodedTestDimensions(t, Dimension{Name: "LoadBalancer", Value: "app/my-lb/50dc6c495c0c9188"}),
			"lookbackPeriod": "1h",
		}
		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		require.True(t, execState.Passed)
		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "Sum", data["statistic"])
		assert.Equal(t, 25.0, data["aggregatedValue"])
	})

	t.Run("no datapoints -> aggregatedValue is nil", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetMetricStatisticsResponse>
						  <GetMetricStatisticsResult>
						    <Datapoints></Datapoints>
						  </GetMetricStatisticsResult>
						</GetMetricStatisticsResponse>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validGetMetricDataConfig(t),
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Nil(t, data["aggregatedValue"])
		assert.Empty(t, data["datapoints"])
	})
}
