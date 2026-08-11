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

// The existing alarm read before the update: SNS actions on both ALARM and OK.
const describeExistingAlarmXML = `
<DescribeAlarmsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <DescribeAlarmsResult>
    <MetricAlarms>
      <member>
        <AlarmName>api-high-cpu</AlarmName>
        <AlarmArn>arn:aws:cloudwatch:us-east-1:123456789012:alarm:api-high-cpu</AlarmArn>
        <AlarmDescription>API instances above 80% CPU</AlarmDescription>
        <ActionsEnabled>true</ActionsEnabled>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Statistic>Average</Statistic>
        <Period>300</Period>
        <EvaluationPeriods>3</EvaluationPeriods>
        <Threshold>80</Threshold>
        <ComparisonOperator>GreaterThanThreshold</ComparisonOperator>
        <StateValue>OK</StateValue>
        <TreatMissingData>missing</TreatMissingData>
        <AlarmActions>
          <member>arn:aws:sns:us-east-1:123456789012:ops-alerts</member>
        </AlarmActions>
        <OKActions>
          <member>arn:aws:sns:us-east-1:123456789012:ops-recovered</member>
        </OKActions>
        <Dimensions>
          <member>
            <Name>InstanceId</Name>
            <Value>i-1234567890abcdef0</Value>
          </member>
        </Dimensions>
      </member>
    </MetricAlarms>
  </DescribeAlarmsResult>
</DescribeAlarmsResponse>`

const describeMetricMathAlarmXML = `
<DescribeAlarmsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <DescribeAlarmsResult>
    <MetricAlarms>
      <member>
        <AlarmName>error-rate</AlarmName>
        <AlarmArn>arn:aws:cloudwatch:us-east-1:123456789012:alarm:error-rate</AlarmArn>
        <ActionsEnabled>true</ActionsEnabled>
        <Threshold>40</Threshold>
        <ComparisonOperator>GreaterThanThreshold</ComparisonOperator>
        <StateValue>OK</StateValue>
        <Metrics>
          <member>
            <Id>error_rate</Id>
          </member>
        </Metrics>
      </member>
    </MetricAlarms>
  </DescribeAlarmsResult>
</DescribeAlarmsResponse>`

func updateAlarmResponses(existing string) []*http.Response {
	return []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(existing))},
		// PutMetricAlarm returns an empty body on success.
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(describeAlarmsXML))},
	}
}

func Test__UpdateAlarm__Setup(t *testing.T) {
	component := &UpdateAlarm{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": " ", "alarm": "api-high-cpu"},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing alarm -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-east-1", "alarm": " "},
		})
		require.ErrorContains(t, err, "alarm name is required")
	})

	t.Run("no property to update -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-east-1", "alarm": "api-high-cpu"},
		})
		require.ErrorContains(t, err, "at least one alarm property to update is required")
	})

	t.Run("threshold without comparison operator -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "api-high-cpu",
				"thresholdCondition": map[string]any{
					"threshold": 90.0,
				},
			},
		})
		require.ErrorContains(t, err, "comparison operator is required")
	})

	t.Run("non-positive period -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "api-high-cpu",
				"period": 0,
			},
		})
		require.ErrorContains(t, err, "period must be greater than 0")
	})

	t.Run("non-positive datapoints to alarm -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"alarm":             "api-high-cpu",
				"datapointsToAlarm": 0,
			},
		})
		require.ErrorContains(t, err, "datapoints to alarm must be greater than 0")
	})

	t.Run("valid configuration -> records updated fields", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "api-high-cpu",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
				"alarmActions": []any{"arn:aws:sns:us-east-1:123456789012:ops-alerts"},
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		nodeMetadata, ok := metadata.Metadata.(UpdateAlarmNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "api-high-cpu", nodeMetadata.AlarmName)
		assert.Equal(t, []string{"Threshold", "Alarm Actions"}, nodeMetadata.UpdatedFields)
	})
}

func Test__UpdateAlarm__Execute(t *testing.T) {
	component := &UpdateAlarm{}

	t.Run("updates alarm and emits alarm details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: updateAlarmResponses(describeExistingAlarmXML)}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "api-high-cpu",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    awsIntegrationContext(),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, UpdateAlarmPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "api-high-cpu", data["alarmName"])
		assert.Equal(t, "AWS/EC2", data["namespace"])

		// The PutMetricAlarm call carries the new threshold plus the untouched
		// identity and actions of the existing alarm.
		body := requestBody(t, httpContext, 1)
		assert.Contains(t, body, "Action=PutMetricAlarm")
		assert.Contains(t, body, "Threshold=90")
		assert.Contains(t, body, "Namespace=AWS%2FEC2")
		assert.Contains(t, body, "MetricName=CPUUtilization")
		assert.Contains(t, body, "Dimensions.member.1.Value=i-1234567890abcdef0")
		assert.Contains(t, body, "AlarmActions.member.1=arn%3Aaws%3Asns%3Aus-east-1%3A123456789012%3Aops-alerts")
		assert.Contains(t, body, "OKActions.member.1=arn%3Aaws%3Asns%3Aus-east-1%3A123456789012%3Aops-recovered")
	})

	t.Run("enabled alarm actions replace the existing ones", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: updateAlarmResponses(describeExistingAlarmXML)}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"alarm":        "api-high-cpu",
				"alarmActions": []any{"arn:aws:sns:us-east-1:123456789012:pager"},
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})
		require.NoError(t, err)

		body := requestBody(t, httpContext, 1)
		assert.Contains(t, body, "AlarmActions.member.1=arn%3Aaws%3Asns%3Aus-east-1%3A123456789012%3Apager")
		assert.NotContains(t, body, "ops-alerts")
		// Untouched transitions keep their actions.
		assert.Contains(t, body, "OKActions.member.1=arn%3Aaws%3Asns%3Aus-east-1%3A123456789012%3Aops-recovered")
	})

	t.Run("enabled but empty alarm actions clear the existing ones", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: updateAlarmResponses(describeExistingAlarmXML)}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"alarm":        "api-high-cpu",
				"alarmActions": []any{},
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})
		require.NoError(t, err)

		body := requestBody(t, httpContext, 1)
		assert.NotContains(t, body, "AlarmActions.member.1")
		assert.Contains(t, body, "OKActions.member.1=arn%3Aaws%3Asns%3Aus-east-1%3A123456789012%3Aops-recovered")
	})

	t.Run("disabling actions keeps the alarm configuration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: updateAlarmResponses(describeExistingAlarmXML)}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":         "us-east-1",
				"alarm":          "api-high-cpu",
				"actionsEnabled": "false",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})
		require.NoError(t, err)

		body := requestBody(t, httpContext, 1)
		assert.Contains(t, body, "ActionsEnabled=false")
		assert.Contains(t, body, "Threshold=80")
	})

	t.Run("metric math alarm -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: updateAlarmResponses(describeMetricMathAlarmXML)}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "error-rate",
				"period": 60,
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "is not a single-metric alarm")
	})

	t.Run("alarm not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
<DescribeAlarmsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <DescribeAlarmsResult><MetricAlarms/></DescribeAlarmsResult>
</DescribeAlarmsResponse>`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "missing-alarm",
				"period": 60,
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "alarm not found: missing-alarm")
	})
}
