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

const deleteAlarmsXML = `
<DeleteAlarmsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <ResponseMetadata>
    <RequestId>0e2b1f5a-3f2f-11f1-9f0a-0f4d9c1c3f11</RequestId>
  </ResponseMetadata>
</DeleteAlarmsResponse>`

const resourceNotFoundXML = `
<ErrorResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <Error>
    <Type>Sender</Type>
    <Code>ResourceNotFound</Code>
    <Message>The named resource does not exist.</Message>
  </Error>
</ErrorResponse>`

func Test__DeleteAlarm__Setup(t *testing.T) {
	component := &DeleteAlarm{}

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
		nodeMetadata, ok := metadata.Metadata.(DeleteAlarmNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", nodeMetadata.Region)
		assert.Equal(t, "api-high-cpu", nodeMetadata.AlarmName)
	})
}

func Test__DeleteAlarm__Execute(t *testing.T) {
	component := &DeleteAlarm{}

	t.Run("deletes the alarm and emits the pre-delete snapshot", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(deleteAlarmsXML)),
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
		assert.Equal(t, DeleteAlarmPayloadType, executionState.Type)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, requestBody(t, httpContext, 0), "Action=DescribeAlarms")

		deleteBody := requestBody(t, httpContext, 1)
		assert.Contains(t, deleteBody, "Action=DeleteAlarms")
		assert.Contains(t, deleteBody, "AlarmNames.member.1=api-high-cpu")

		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, data["deleted"])
		assert.Equal(t, "api-high-cpu", data["alarmName"])
		assert.Equal(t, "arn:aws:cloudwatch:us-east-1:123456789012:alarm:api-high-cpu", data["alarmArn"])
		assert.Equal(t, "AWS/EC2", data["namespace"])
		assert.Equal(t, "CPUUtilization", data["metricName"])
		assert.Equal(t, float64(80), data["threshold"])
		assert.Equal(t, "GreaterThanThreshold", data["comparisonOperator"])
		assert.Equal(t, "us-east-1", data["region"])

		// The console URL is dropped: it no longer resolves once the alarm is gone.
		assert.NotContains(t, data, "consoleUrl")
	})

	// A metric math alarm still describes as a metric alarm, so it is deleted. Its
	// metric definition is not in the snapshot, which the documentation calls out.
	t.Run("metric math alarm -> deleted with an incomplete snapshot", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeMetricMathAlarmXML)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(deleteAlarmsXML)),
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
		assert.Contains(t, requestBody(t, httpContext, 1), "AlarmNames.member.1=error-rate")

		require.Len(t, executionState.Payloads, 1)
		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, true, data["deleted"])
		assert.Equal(t, "error-rate", data["alarmName"])
		assert.Empty(t, data["namespace"])
		assert.Empty(t, data["metricName"])
	})

	t.Run("unknown alarm -> error and no delete call", func(t *testing.T) {
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
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("delete rejected by CloudWatch -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(describeAlarmsXML)),
				},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(resourceNotFoundXML)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"alarm":  "api-high-cpu",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{},
			Integration:    awsIntegrationContext(),
		})

		require.ErrorContains(t, err, "failed to delete alarm")
		require.ErrorContains(t, err, "The named resource does not exist.")
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

func Test__DeleteAlarms__Client(t *testing.T) {
	t.Run("blank alarm names -> error", func(t *testing.T) {
		client := NewClient(nil, nil, "us-east-1")
		require.ErrorContains(t, client.DeleteAlarms("  ", ""), "at least one alarm name is required")
	})
}
