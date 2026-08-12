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

const describeAlarmsXML = `
<DescribeAlarmsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <DescribeAlarmsResult>
    <MetricAlarms>
      <member>
        <AlarmName>api-high-cpu</AlarmName>
        <AlarmArn>arn:aws:cloudwatch:us-east-1:123456789012:alarm:api-high-cpu</AlarmArn>
        <AlarmDescription>API instances above 80% CPU</AlarmDescription>
        <AlarmConfigurationUpdatedTimestamp>2026-06-03T16:36:13.412Z</AlarmConfigurationUpdatedTimestamp>
        <ActionsEnabled>true</ActionsEnabled>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Statistic>Average</Statistic>
        <Unit>Percent</Unit>
        <Period>300</Period>
        <EvaluationPeriods>3</EvaluationPeriods>
        <DatapointsToAlarm>2</DatapointsToAlarm>
        <Threshold>80</Threshold>
        <ComparisonOperator>GreaterThanThreshold</ComparisonOperator>
        <StateValue>INSUFFICIENT_DATA</StateValue>
        <StateReason>Unchecked: Initial alarm creation</StateReason>
        <TreatMissingData>missing</TreatMissingData>
        <AlarmActions>
          <member>arn:aws:sns:us-east-1:123456789012:ops-alerts</member>
        </AlarmActions>
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

const instanceDimensions = `[{"name":"InstanceId","value":"i-1234567890abcdef0"}]`

func awsIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
			"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
			"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
		},
	}
}

func Test__CreateAlarm__Setup(t *testing.T) {
	component := &CreateAlarm{}

	validConfiguration := func() map[string]any {
		return map[string]any{
			"region":             "us-east-1",
			"alarmName":          "api-high-cpu",
			"namespace":          "AWS/EC2",
			"metricName":         "CPUUtilization",
			"dimensions":         instanceDimensions,
			"statistic":          "Average",
			"comparisonOperator": "GreaterThanThreshold",
			"threshold":          80.0,
		}
	}

	t.Run("missing region -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["region"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing alarm name -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["alarmName"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "alarm name is required")
	})

	t.Run("missing namespace -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["namespace"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("missing metric name -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["metricName"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "metric name is required")
	})

	t.Run("missing statistic -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["statistic"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "statistic is required")
	})

	t.Run("missing comparison operator -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["comparisonOperator"] = " "
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "comparison operator is required")
	})

	t.Run("missing threshold -> error", func(t *testing.T) {
		configuration := validConfiguration()
		delete(configuration, "threshold")
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "threshold is required")
	})

	t.Run("malformed dimensions -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["dimensions"] = "InstanceId=i-123"
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "invalid dimensions")
	})

	t.Run("datapoints to alarm above evaluation periods -> error", func(t *testing.T) {
		configuration := validConfiguration()
		configuration["evaluationPeriods"] = 3
		configuration["datapointsToAlarm"] = 5
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "datapoints to alarm (5) cannot be greater than evaluation periods (3)")
	})

	t.Run("datapoints to alarm above the default evaluation periods -> error", func(t *testing.T) {
		// evaluationPeriods left empty defaults to 1, so 2 of 1 is still invalid.
		configuration := validConfiguration()
		configuration["datapointsToAlarm"] = 2
		err := component.Setup(core.SetupContext{Configuration: configuration})
		require.ErrorContains(t, err, "cannot be greater than evaluation periods (1)")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: validConfiguration(),
			Metadata:      metadata,
		})
		require.NoError(t, err)

		nodeMetadata, ok := metadata.Metadata.(CreateAlarmNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "api-high-cpu", nodeMetadata.AlarmName)
		assert.Equal(t, "AWS/EC2", nodeMetadata.Namespace)
		assert.Equal(t, "CPUUtilization", nodeMetadata.MetricName)
		assert.Equal(t, "InstanceId=i-1234567890abcdef0", nodeMetadata.Dimensions)
	})
}

func Test__CreateAlarm__Execute(t *testing.T) {
	component := &CreateAlarm{}

	t.Run("creates alarm and emits alarm details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// PutMetricAlarm returns an empty body on success.
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(describeAlarmsXML))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"alarmName":          "api-high-cpu",
				"namespace":          "AWS/EC2",
				"metricName":         "CPUUtilization",
				"dimensions":         instanceDimensions,
				"statistic":          "Average",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          80.0,
				"period":             300,
				"evaluationPeriods":  3,
				"datapointsToAlarm":  2,
				"alarmActions":       []any{"arn:aws:sns:us-east-1:123456789012:ops-alerts"},
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    awsIntegrationContext(),
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
		assert.Equal(t, "api-high-cpu", data["alarmName"])
		assert.Equal(t, "AWS/EC2", data["namespace"])
		assert.Equal(t, "CPUUtilization", data["metricName"])
		assert.Equal(t, "INSUFFICIENT_DATA", data["stateValue"])
		assert.Equal(t, float64(80), data["threshold"])
		assert.Equal(t, 2, data["datapointsToAlarm"])
		assert.Equal(t, true, data["actionsEnabled"])
		assert.Equal(t, []string{"arn:aws:sns:us-east-1:123456789012:ops-alerts"}, data["alarmActions"])
		assert.Equal(
			t,
			"https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
			data["consoleUrl"],
		)
	})

	t.Run("sends the configured metric identity to CloudWatch", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(describeAlarmsXML))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"alarmName":          "api-high-cpu",
				"namespace":          "AWS/EC2",
				"metricName":         "CPUUtilization",
				"dimensions":         instanceDimensions,
				"statistic":          "Average",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          80.0,
				"unit":               "Percent",
				"treatMissingData":   "missing",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})
		require.NoError(t, err)

		require.NotEmpty(t, httpContext.Requests)
		body := requestBody(t, httpContext, 0)
		assert.Contains(t, body, "Action=PutMetricAlarm")
		assert.Contains(t, body, "Namespace=AWS%2FEC2")
		assert.Contains(t, body, "MetricName=CPUUtilization")
		assert.Contains(t, body, "Dimensions.member.1.Name=InstanceId")
		assert.Contains(t, body, "Dimensions.member.1.Value=i-1234567890abcdef0")
		assert.Contains(t, body, "Unit=Percent")
		assert.Contains(t, body, "TreatMissingData=missing")
		assert.Contains(t, body, "ActionsEnabled=true")
	})

	t.Run("ec2 action is sent alongside the sns topics", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(describeAlarmsXML))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"alarmName":          "api-status-check",
				"namespace":          "AWS/EC2",
				"metricName":         "StatusCheckFailed_System",
				"dimensions":         instanceDimensions,
				"statistic":          "Maximum",
				"comparisonOperator": "GreaterThanOrEqualToThreshold",
				"threshold":          1.0,
				"alarmActions":       []any{"arn:aws:sns:us-east-1:123456789012:ops-alerts"},
				"ec2Action":          "recover",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})
		require.NoError(t, err)

		body := requestBody(t, httpContext, 0)
		assert.Contains(t, body, "AlarmActions.member.1=arn%3Aaws%3Asns%3Aus-east-1%3A123456789012%3Aops-alerts")
		assert.Contains(t, body, "AlarmActions.member.2=arn%3Aaws%3Aautomate%3Aus-east-1%3Aec2%3Arecover")
	})

	t.Run("ec2 action on a non-EC2 namespace -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"alarmName":          "queue-depth",
				"namespace":          "AWS/SQS",
				"metricName":         "ApproximateNumberOfMessagesVisible",
				"statistic":          "Maximum",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          100.0,
				"ec2Action":          "reboot",
			},
			HTTP:           &contexts.HTTPContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "EC2 actions require an AWS/EC2 alarm")
	})

	t.Run("ec2 action without an InstanceId dimension -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"alarmName":          "fleet-cpu",
				"namespace":          "AWS/EC2",
				"metricName":         "CPUUtilization",
				"dimensions":         `[{"name":"AutoScalingGroupName","value":"api-asg"}]`,
				"statistic":          "Average",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          80.0,
				"ec2Action":          "stop",
			},
			HTTP:           &contexts.HTTPContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "require the alarm to have an InstanceId dimension")
	})

	t.Run("alarm creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
<ErrorResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <Error>
    <Code>ValidationError</Code>
    <Message>Invalid metric name</Message>
  </Error>
</ErrorResponse>`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":             "us-east-1",
				"alarmName":          "api-high-cpu",
				"namespace":          "AWS/EC2",
				"metricName":         "CPUUtilization",
				"statistic":          "Average",
				"comparisonOperator": "GreaterThanThreshold",
				"threshold":          80.0,
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to create alarm")
		require.ErrorContains(t, err, "Invalid metric name")
	})
}

func requestBody(t *testing.T, httpContext *contexts.HTTPContext, index int) string {
	t.Helper()

	require.Greater(t, len(httpContext.Requests), index)
	body, err := io.ReadAll(httpContext.Requests[index].Body)
	require.NoError(t, err)
	return string(body)
}
