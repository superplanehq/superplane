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

const describeMissingAlarmXML = `
<DescribeAlarmsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <DescribeAlarmsResult>
    <MetricAlarms/>
  </DescribeAlarmsResult>
</DescribeAlarmsResponse>`

func Test__GetAlarm__Setup(t *testing.T) {
	component := &GetAlarm{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": " ",
				"alarm":  "api-high-cpu",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing alarm name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  " ",
			},
		})
		require.ErrorContains(t, err, "alarm name is required")
	})

	t.Run("valid configuration -> metadata is set", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "  api-high-cpu  ",
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		nodeMetadata, ok := metadata.Metadata.(GetAlarmNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.Equal(t, "api-high-cpu", nodeMetadata.AlarmName)
	})
}

func Test__GetAlarm__Execute(t *testing.T) {
	component := &GetAlarm{}

	t.Run("describes the alarm and emits it", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "api-high-cpu",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    awsIntegrationContext(),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, GetAlarmPayloadType, executionState.Type)

		body := requestBody(t, httpContext, 0)
		assert.Contains(t, body, "Action=DescribeAlarms")
		assert.Contains(t, body, "AlarmNames.member.1=api-high-cpu")

		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "api-high-cpu", data["alarmName"])
		assert.Equal(t, "arn:aws:cloudwatch:us-east-1:123456789012:alarm:api-high-cpu", data["alarmArn"])
		assert.Equal(t, "AWS/EC2", data["namespace"])
		assert.Equal(t, "CPUUtilization", data["metricName"])
		assert.Equal(t, "Average", data["statistic"])
		assert.Equal(t, "Percent", data["unit"])
		assert.Equal(t, 300, data["period"])
		assert.Equal(t, 3, data["evaluationPeriods"])
		assert.Equal(t, 2, data["datapointsToAlarm"])
		assert.Equal(t, float64(80), data["threshold"])
		assert.Equal(t, "GreaterThanThreshold", data["comparisonOperator"])
		assert.Equal(t, "INSUFFICIENT_DATA", data["stateValue"])
		assert.Equal(t, "missing", data["treatMissingData"])
		assert.Equal(t, true, data["actionsEnabled"])
		assert.Equal(t, []string{"arn:aws:sns:us-east-1:123456789012:ops-alerts"}, data["alarmActions"])
		assert.Equal(t, "us-east-1", data["region"])
		assert.Equal(
			t,
			"https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
			data["consoleUrl"],
		)
		assert.Equal(
			t,
			[]map[string]any{{"name": "InstanceId", "value": "i-1234567890abcdef0"}},
			data["dimensions"],
		)
	})

	t.Run("console URL follows the region partition", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "cn-north-1",
				"alarm":  "api-high-cpu",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    awsIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(
			t,
			"https://cn-north-1.console.amazonaws.cn/cloudwatch/home?region=cn-north-1#alarmsV2:alarm/api-high-cpu",
			data["consoleUrl"],
		)
	})

	// A metric math alarm still describes as a metric alarm, so it reaches the
	// component and emits with the single-metric fields empty, as documented.
	t.Run("metric math alarm -> emitted without a metric definition", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeMetricMathAlarmXML)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "error-rate",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    awsIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "error-rate", data["alarmName"])
		assert.Equal(t, "OK", data["stateValue"])
		assert.Empty(t, data["namespace"])
		assert.Empty(t, data["metricName"])
		assert.Empty(t, data["statistic"])
		assert.Equal(t, 0, data["period"])
	})

	t.Run("unknown alarm -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeMissingAlarmXML)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "does-not-exist",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to describe alarm")
		require.ErrorContains(t, err, "alarm not found: does-not-exist")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": " ",
				"alarm":  "api-high-cpu",
			},
			HTTP:           &contexts.HTTPContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing alarm name -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  " ",
			},
			HTTP:           &contexts.HTTPContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "alarm name is required")
	})
}
