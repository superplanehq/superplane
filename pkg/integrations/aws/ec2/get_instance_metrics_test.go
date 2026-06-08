package ec2

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetInstanceMetrics__Setup(t *testing.T) {
	component := &GetInstanceMetrics{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         " ",
				"instance":       "i-abc123",
				"lookbackPeriod": "1h",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing instance -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "",
				"lookbackPeriod": "1h",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("missing lookbackPeriod -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "i-abc123",
				"lookbackPeriod": "",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "lookbackPeriod is required")
	})

	t.Run("invalid lookbackPeriod -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "i-abc123",
				"lookbackPeriod": "30m",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid lookbackPeriod")
	})

	t.Run("valid config stores metadata", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(describeInstanceXML("i-abc123", "running")),
			},
		}
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "i-abc123",
				"lookbackPeriod": "1h",
			},
			HTTP:        httpCtx,
			Integration: metricsIntegration(),
			Metadata:    metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(GetInstanceMetricsNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "i-abc123", stored.InstanceID)
	})
}

func Test__GetInstanceMetrics__Execute(t *testing.T) {
	component := &GetInstanceMetrics{}

	t.Run("emits cpu and network metrics without memory", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Three concurrent calls: CPU, NetworkIn, NetworkOut
				okResponse(cloudWatchMetricsXML("Average", "12.5")),
				okResponse(cloudWatchNetworkXML("Sum", "1024000")),
				okResponse(cloudWatchNetworkXML("Sum", "512000")),
			},
		}
		execState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "i-abc123",
				"lookbackPeriod": "1h",
				"includeMemory":  false,
			},
			HTTP:           httpCtx,
			Integration:    metricsIntegration(),
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, GetInstanceMetricsPayloadType, execState.Type)
		require.NotEmpty(t, execState.Payloads)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "i-abc123", payload["instanceId"])
		assert.Equal(t, "us-east-1", payload["region"])
		assert.Equal(t, "1h", payload["lookbackPeriod"])
		assert.NotEmpty(t, payload["start"])
		assert.NotEmpty(t, payload["end"])
		_, hasMemory := payload["avgMemoryUsagePercent"]
		assert.False(t, hasMemory)
	})

	t.Run("emits metrics with memory when includeMemory=true and data available", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(cloudWatchMetricsXML("Average", "20.0")),
				okResponse(cloudWatchNetworkXML("Sum", "2048000")),
				okResponse(cloudWatchNetworkXML("Sum", "1024000")),
				okResponse(cloudWatchMetricsXML("Average", "65.5")),
			},
		}
		execState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "i-abc123",
				"lookbackPeriod": "1h",
				"includeMemory":  true,
			},
			HTTP:           httpCtx,
			Integration:    metricsIntegration(),
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		require.NotEmpty(t, execState.Payloads)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		_, hasMemory := payload["avgMemoryUsagePercent"]
		assert.True(t, hasMemory)
	})

	t.Run("memory null when agent returns no datapoints", func(t *testing.T) {
		// All goroutines share the same mock HTTP context and consume responses in
		// non-deterministic order. Using empty metric responses for all four calls
		// ensures that regardless of ordering, every metric (including memory) has
		// zero datapoints, making avgMemoryUsagePercent reliably nil.
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(cloudWatchEmptyMetricsXML()),
				okResponse(cloudWatchEmptyMetricsXML()),
				okResponse(cloudWatchEmptyMetricsXML()),
				okResponse(cloudWatchEmptyMetricsXML()),
			},
		}
		execState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "i-abc123",
				"lookbackPeriod": "1h",
				"includeMemory":  true,
			},
			HTTP:           httpCtx,
			Integration:    metricsIntegration(),
			ExecutionState: execState,
		})

		require.NoError(t, err)
		require.NotEmpty(t, execState.Payloads)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		val, hasMemory := payload["avgMemoryUsagePercent"]
		assert.True(t, hasMemory)
		assert.Nil(t, val)

		cpuVal, hasCPU := payload["avgCpuUsagePercent"]
		assert.True(t, hasCPU)
		assert.Nil(t, cpuVal)
	})

	t.Run("cpu null when cloudwatch returns no datapoints", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(cloudWatchEmptyMetricsXML()),
				okResponse(cloudWatchEmptyMetricsXML()),
				okResponse(cloudWatchEmptyMetricsXML()),
			},
		}
		execState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"instance":       "i-abc123",
				"lookbackPeriod": "1h",
				"includeMemory":  false,
			},
			HTTP:           httpCtx,
			Integration:    metricsIntegration(),
			ExecutionState: execState,
		})

		require.NoError(t, err)
		require.NotEmpty(t, execState.Payloads)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		val, hasCPU := payload["avgCpuUsagePercent"]
		assert.True(t, hasCPU)
		assert.Nil(t, val)
	})
}

func Test__memoryUsagePercent(t *testing.T) {
	t.Run("returns nil when cloudwatch request fails", func(t *testing.T) {
		assert.Nil(t, memoryUsagePercent(errors.New("access denied"), nil))
	})

	t.Run("returns nil when no datapoints", func(t *testing.T) {
		assert.Nil(t, memoryUsagePercent(nil, nil))
	})

	t.Run("returns average when datapoints exist", func(t *testing.T) {
		assert.Equal(t, 65.5, memoryUsagePercent(nil, []CloudWatchDatapoint{{Average: 65.5}}))
	})
}

// Helpers

func metricsIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func cloudWatchMetricsXML(statistic, value string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<GetMetricStatisticsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetMetricStatisticsResult>
    <Label>CPUUtilization</Label>
    <Datapoints>
      <member>
        <Timestamp>2024-04-01T12:00:00Z</Timestamp>
        <` + statistic + `>` + value + `</` + statistic + `>
        <Unit>Percent</Unit>
      </member>
    </Datapoints>
  </GetMetricStatisticsResult>
  <ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata>
</GetMetricStatisticsResponse>`
}

func cloudWatchNetworkXML(statistic, value string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<GetMetricStatisticsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetMetricStatisticsResult>
    <Label>NetworkIn</Label>
    <Datapoints>
      <member>
        <Timestamp>2024-04-01T12:00:00Z</Timestamp>
        <` + statistic + `>` + value + `</` + statistic + `>
        <Unit>Bytes</Unit>
      </member>
    </Datapoints>
  </GetMetricStatisticsResult>
  <ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata>
</GetMetricStatisticsResponse>`
}

func cloudWatchEmptyMetricsXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<GetMetricStatisticsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetMetricStatisticsResult>
    <Label>mem_used_percent</Label>
    <Datapoints/>
  </GetMetricStatisticsResult>
  <ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata>
</GetMetricStatisticsResponse>`
}
