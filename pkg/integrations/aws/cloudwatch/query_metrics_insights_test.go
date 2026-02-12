package cloudwatch

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__QueryMetricsInsights__Setup(t *testing.T) {
	component := &QueryMetricsInsights{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId)",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing query -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})

		require.ErrorContains(t, err, "metrics insights query is required")
	})

	t.Run("invalid scanBy -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"query":  "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId)",
				"scanBy": "descending",
			},
		})

		require.ErrorContains(t, err, "invalid scan by value")
	})

	t.Run("negative lookback -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"query":           "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId)",
				"lookbackMinutes": -5,
			},
		})

		require.ErrorContains(t, err, "lookback minutes must be greater than or equal to zero")
	})

	t.Run("max datapoints above allowed limit -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":        "us-east-1",
				"query":         "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId)",
				"maxDatapoints": maxAllowedCloudWatchDatapoints + 1,
			},
		})

		require.ErrorContains(t, err, "max datapoints must be less than or equal to")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"query":           "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId)",
				"lookbackMinutes": 10,
				"maxDatapoints":   500,
				"scanBy":          ScanByTimestampDescending,
			},
		})

		require.NoError(t, err)
	})
}

func Test__QueryMetricsInsights__Execute(t *testing.T) {
	component := &QueryMetricsInsights{}

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"query":  "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId)",
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("paginated response -> emits merged metric results", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(getMetricDataResponsePageOne())),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(getMetricDataResponsePageTwo())),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":          "us-east-1",
				"query":           "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId) GROUP BY InstanceId",
				"lookbackMinutes": 20,
				"maxDatapoints":   250,
				"scanBy":          ScanByTimestampDescending,
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration: &contexts.IntegrationContext{
				Secrets: awsSessionSecrets(),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, QueryMetricsInsightsEventType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "us-east-1", payload["region"])
		assert.Equal(t, 250, payload["maxDatapoints"])
		assert.Equal(t, ScanByTimestampDescending, payload["scanBy"])
		assert.Equal(t, "request-2", payload["requestId"])

		startTimeValue, ok := payload["startTime"].(string)
		require.True(t, ok)
		endTimeValue, ok := payload["endTime"].(string)
		require.True(t, ok)
		startTime, err := time.Parse(time.RFC3339, startTimeValue)
		require.NoError(t, err)
		endTime, err := time.Parse(time.RFC3339, endTimeValue)
		require.NoError(t, err)
		assert.True(t, endTime.After(startTime))

		results, ok := payload["results"].([]MetricDataResult)
		require.True(t, ok)
		require.Len(t, results, 1)
		assert.Equal(t, "q1", results[0].ID)
		assert.Equal(t, "CPUUtilization", results[0].Label)
		assert.Equal(t, "Complete", results[0].StatusCode)
		assert.Equal(t, []string{"2026-02-12T10:10:00Z", "2026-02-12T10:05:00Z"}, results[0].Timestamps)
		assert.Equal(t, []float64{10.5, 11.5}, results[0].Values)

		messages, ok := payload["messages"].([]MetricDataMessage)
		require.True(t, ok)
		require.Len(t, messages, 1)
		assert.Equal(t, "Info", messages[0].Code)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://monitoring.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://monitoring.us-east-1.amazonaws.com/", httpContext.Requests[1].URL.String())

		firstRequestBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		firstValues, err := url.ParseQuery(string(firstRequestBody))
		require.NoError(t, err)
		assert.Equal(t, "GetMetricData", firstValues.Get("Action"))
		assert.Equal(t, "2010-08-01", firstValues.Get("Version"))
		assert.Equal(t, "q1", firstValues.Get("MetricDataQueries.member.1.Id"))
		assert.Equal(t, "60", firstValues.Get("MetricDataQueries.member.1.Period"))
		assert.Equal(t, "", firstValues.Get("NextToken"))

		secondRequestBody, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		secondValues, err := url.ParseQuery(string(secondRequestBody))
		require.NoError(t, err)
		assert.Equal(t, "token-1", secondValues.Get("NextToken"))
	})
}

func awsSessionSecrets() map[string]core.IntegrationSecret {
	return map[string]core.IntegrationSecret{
		"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
		"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
		"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
	}
}

func getMetricDataResponsePageOne() string {
	return `
<GetMetricDataResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetMetricDataResult>
    <MetricDataResults>
      <member>
        <Id>q1</Id>
        <Label>CPUUtilization</Label>
        <StatusCode>PartialData</StatusCode>
        <Timestamps>
          <member>2026-02-12T10:10:00Z</member>
        </Timestamps>
        <Values>
          <member>10.5</member>
        </Values>
      </member>
    </MetricDataResults>
    <NextToken>token-1</NextToken>
  </GetMetricDataResult>
  <ResponseMetadata>
    <RequestId>request-1</RequestId>
  </ResponseMetadata>
</GetMetricDataResponse>
`
}

func getMetricDataResponsePageTwo() string {
	return `
<GetMetricDataResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetMetricDataResult>
    <MetricDataResults>
      <member>
        <Id>q1</Id>
        <Label>CPUUtilization</Label>
        <StatusCode>Complete</StatusCode>
        <Timestamps>
          <member>2026-02-12T10:05:00Z</member>
        </Timestamps>
        <Values>
          <member>11.5</member>
        </Values>
      </member>
    </MetricDataResults>
    <Messages>
      <member>
        <Code>Info</Code>
        <Value>Query completed successfully</Value>
      </member>
    </Messages>
  </GetMetricDataResult>
  <ResponseMetadata>
    <RequestId>request-2</RequestId>
  </ResponseMetadata>
</GetMetricDataResponse>
`
}
