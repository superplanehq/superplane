package ec2

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// describeAlarmsWithActionsXML has an existing SNS topic and EC2 automation action.
const describeAlarmsWithActionsXML = `
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
        <AlarmActions>
          <member>arn:aws:automate:us-east-1:ec2:recover</member>
          <member>arn:aws:sns:us-east-1:123456789012:existing-topic</member>
        </AlarmActions>
      </member>
    </MetricAlarms>
  </DescribeAlarmsResult>
</DescribeAlarmsResponse>`

func Test__UpdateAlarm__Setup(t *testing.T) {
	component := &UpdateAlarm{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": " ",
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing alarm name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  " ",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
		})
		require.ErrorContains(t, err, "alarm name is required")
	})

	t.Run("no update fields -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
			},
		})
		require.ErrorContains(t, err, "at least one alarm property to update is required")
	})

	t.Run("threshold condition missing threshold -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"comparisonOperator": "GreaterThanThreshold",
				},
			},
		})
		require.ErrorContains(t, err, "threshold is required")
	})

	t.Run("valid configuration -> stores updated fields in metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
				"statistic": "Average",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		stored, ok := metadata.Get().(UpdateAlarmNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, []string{"Threshold", "Statistic"}, stored.UpdatedFields)
	})

	t.Run("period = 0 -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "HighCPU",
				"period": 0,
			},
		})
		require.ErrorContains(t, err, "period must be greater than 0")
	})

	t.Run("evaluationPeriods = 0 -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"alarm":             "HighCPU",
				"evaluationPeriods": 0,
			},
		})
		require.ErrorContains(t, err, "evaluation periods must be greater than 0")
	})
}

func Test__UpdateAlarm__Execute(t *testing.T) {
	component := &UpdateAlarm{}

	t.Run("updates alarm and emits alarm details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
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
				"alarm":  "HighCPU",
				"thresholdCondition": map[string]any{
					"threshold":          90.0,
					"comparisonOperator": "GreaterThanThreshold",
				},
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
		assert.Equal(t, UpdateAlarmPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "HighCPU", data["alarmName"])
		assert.Equal(t, float64(80), data["threshold"])
	})

	t.Run("updating only alarmAction preserves existing SNS topic", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"alarm":       "HighCPU",
				"alarmAction": "reboot",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)

		// Inspect the PutMetricAlarm request body (second HTTP call).
		require.Len(t, httpContext.Requests, 3)
		putBody, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		params, err := url.ParseQuery(string(putBody))
		require.NoError(t, err)

		// Both the new EC2 action and the preserved SNS topic must be present.
		actions := []string{params.Get("AlarmActions.member.1"), params.Get("AlarmActions.member.2")}
		assert.Contains(t, actions, "arn:aws:automate:us-east-1:ec2:reboot")
		assert.Contains(t, actions, "arn:aws:sns:us-east-1:123456789012:existing-topic")
	})

	t.Run("updating only snsTopic preserves existing EC2 automation action", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":   "us-east-1",
				"alarm":    "HighCPU",
				"snsTopic": "arn:aws:sns:us-east-1:123456789012:new-topic",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 3)
		putBody, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		params, err := url.ParseQuery(string(putBody))
		require.NoError(t, err)

		// New SNS topic should be present.
		actions := []string{params.Get("AlarmActions.member.1"), params.Get("AlarmActions.member.2")}
		assert.Contains(t, actions, "arn:aws:sns:us-east-1:123456789012:new-topic")
		// Existing EC2 automation action must not have been dropped.
		assert.Contains(t, actions, "arn:aws:automate:us-east-1:ec2:recover")
	})

	t.Run("clearing alarm description sends empty AlarmDescription", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":           "us-east-1",
				"alarm":            "HighCPU",
				"alarmDescription": "",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 3)
		putBody, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		params, err := url.ParseQuery(string(putBody))
		require.NoError(t, err)

		_, hasDescription := params["AlarmDescription"]
		assert.True(t, hasDescription, "AlarmDescription must be sent to clear an existing description")
		assert.Empty(t, params.Get("AlarmDescription"))
	})

	t.Run("clearing both alarm action and SNS topic sends no AlarmActions members", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"alarm":       "HighCPU",
				"alarmAction": "",
				"snsTopic":    "",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 3)
		putBody, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		params, err := url.ParseQuery(string(putBody))
		require.NoError(t, err)

		// No AlarmActions members should be present — empty list signals CloudWatch to clear them.
		assert.Empty(t, params.Get("AlarmActions.member.1"), "clearing both action fields must send no AlarmActions members")
	})

	t.Run("not toggling alarm actions re-sends existing actions to preserve them", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsWithActionsXML)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"alarm":     "HighCPU",
				"statistic": "Sum",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 3)
		putBody, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		params, err := url.ParseQuery(string(putBody))
		require.NoError(t, err)

		// Existing actions must be re-sent explicitly to preserve them.
		actions := []string{params.Get("AlarmActions.member.1"), params.Get("AlarmActions.member.2")}
		assert.Contains(t, actions, "arn:aws:automate:us-east-1:ec2:recover")
		assert.Contains(t, actions, "arn:aws:sns:us-east-1:123456789012:existing-topic")
	})
}
