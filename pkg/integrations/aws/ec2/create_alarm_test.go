package ec2

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

const describeAlarmsXML = `
<DescribeAlarmsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <DescribeAlarmsResult>
    <MetricAlarms>
      <member>
        <AlarmName>HighCPU</AlarmName>
        <AlarmArn>arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighCPU</AlarmArn>
        <AlarmDescription>High CPU utilization alarm</AlarmDescription>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Statistic>Average</Statistic>
        <Period>300</Period>
        <EvaluationPeriods>1</EvaluationPeriods>
        <Threshold>80</Threshold>
        <ComparisonOperator>GreaterThanThreshold</ComparisonOperator>
        <StateValue>OK</StateValue>
        <StateReason>Threshold Crossed: no datapoints</StateReason>
        <TreatMissingData>missing</TreatMissingData>
        <Dimensions>
          <member>
            <Name>InstanceId</Name>
            <Value>i-abc123</Value>
          </member>
        </Dimensions>
      </member>
    </MetricAlarms>
  </DescribeAlarmsResult>
</DescribeAlarmsResponse>`

func Test__CreateAlarm__Setup(t *testing.T) {
	component := &CreateAlarm{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             " ",
				"instance":           "i-abc123",
				"alarmName":          "HighCPU",
				"metricName":         "CPUUtilization",
				"comparisonOperator": "GreaterThanThreshold",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing instance -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           " ",
				"alarmName":          "HighCPU",
				"metricName":         "CPUUtilization",
				"comparisonOperator": "GreaterThanThreshold",
			},
		})
		require.ErrorContains(t, err, "instance ID is required")
	})

	t.Run("missing alarm name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"alarmName":          " ",
				"metricName":         "CPUUtilization",
				"comparisonOperator": "GreaterThanThreshold",
			},
		})
		require.ErrorContains(t, err, "alarm name is required")
	})

	t.Run("missing metric name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"alarmName":          "HighCPU",
				"metricName":         " ",
				"comparisonOperator": "GreaterThanThreshold",
			},
		})
		require.ErrorContains(t, err, "metric name is required")
	})

	t.Run("missing comparison operator -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"alarmName":          "HighCPU",
				"metricName":         "CPUUtilization",
				"statistic":          "Average",
				"comparisonOperator": " ",
				"threshold":          80.0,
			},
		})
		require.ErrorContains(t, err, "comparison operator is required")
	})

	t.Run("missing statistic -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"alarmName":          "HighCPU",
				"metricName":         "CPUUtilization",
				"statistic":          " ",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          80.0,
			},
		})
		require.ErrorContains(t, err, "statistic is required")
	})

	t.Run("missing threshold -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"alarmName":          "HighCPU",
				"metricName":         "CPUUtilization",
				"statistic":          "Average",
				"comparisonOperator": "GreaterThanThreshold",
			},
		})
		require.ErrorContains(t, err, "threshold is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"alarmName":          "HighCPU",
				"metricName":         "CPUUtilization",
				"statistic":          "Average",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          80.0,
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__CreateAlarm__Execute(t *testing.T) {
	component := &CreateAlarm{}

	t.Run("creates alarm and emits alarm details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// PutMetricAlarm response (empty body, 200)
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				// DescribeAlarms response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"instance":           "i-abc123",
				"alarmName":          "HighCPU",
				"metricName":         "CPUUtilization",
				"statistic":          "Average",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          80.0,
				"period":             300,
				"evaluationPeriods":  1,
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, CreateAlarmPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "HighCPU", data["alarmName"])
		assert.Equal(t, "CPUUtilization", data["metricName"])
		assert.Equal(t, "AWS/EC2", data["namespace"])
		assert.Equal(t, "OK", data["stateValue"])
		assert.Equal(t, float64(80), data["threshold"])
	})
}
