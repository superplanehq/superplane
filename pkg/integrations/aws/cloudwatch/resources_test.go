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

const listMetricsXML = `
<ListMetricsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <ListMetricsResult>
    <Metrics>
      <member>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Dimensions>
          <member><Name>InstanceId</Name><Value>i-1234567890abcdef0</Value></member>
        </Dimensions>
      </member>
      <member>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Dimensions>
          <member><Name>InstanceId</Name><Value>i-abcdef01234567890</Value></member>
        </Dimensions>
      </member>
      <member>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>NetworkIn</MetricName>
        <Dimensions>
          <member><Name>InstanceId</Name><Value>i-1234567890abcdef0</Value></member>
        </Dimensions>
      </member>
      <member>
        <Namespace>AWS/Lambda</Namespace>
        <MetricName>Errors</MetricName>
        <Dimensions>
          <member><Name>FunctionName</Name><Value>checkout</Value></member>
        </Dimensions>
      </member>
    </Metrics>
  </ListMetricsResult>
</ListMetricsResponse>`

// What ListMetrics returns once CloudWatch applies the Namespace/MetricName filters.
const listEC2CPUMetricsXML = `
<ListMetricsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <ListMetricsResult>
    <Metrics>
      <member>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Dimensions>
          <member><Name>InstanceId</Name><Value>i-1234567890abcdef0</Value></member>
        </Dimensions>
      </member>
      <member>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Dimensions>
          <member><Name>InstanceId</Name><Value>i-abcdef01234567890</Value></member>
        </Dimensions>
      </member>
      <member>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>NetworkIn</MetricName>
        <Dimensions>
          <member><Name>InstanceId</Name><Value>i-1234567890abcdef0</Value></member>
        </Dimensions>
      </member>
      <member>
        <Namespace>AWS/EC2</Namespace>
        <MetricName>CPUUtilization</MetricName>
        <Dimensions>
          <member><Name>AutoScalingGroupName</Name><Value>api-asg</Value></member>
        </Dimensions>
      </member>
    </Metrics>
  </ListMetricsResult>
</ListMetricsResponse>`

func listResourcesContext(responses ...string) (core.ListResourcesContext, *contexts.HTTPContext) {
	httpResponses := make([]*http.Response, 0, len(responses))
	for _, response := range responses {
		httpResponses = append(httpResponses, &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(response)),
		})
	}

	httpContext := &contexts.HTTPContext{Responses: httpResponses}
	return core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: awsIntegrationContext(),
		Parameters:  map[string]string{"region": "us-east-1"},
	}, httpContext
}

func Test__ListAlarms(t *testing.T) {
	t.Run("missing region -> error", func(t *testing.T) {
		ctx, _ := listResourcesContext()
		ctx.Parameters = map[string]string{}

		_, err := ListAlarms(ctx, "cloudwatch.alarm")
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("returns alarms by name", func(t *testing.T) {
		ctx, httpContext := listResourcesContext(describeAlarmsXML)

		resources, err := ListAlarms(ctx, "cloudwatch.alarm")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "cloudwatch.alarm", resources[0].Type)
		assert.Equal(t, "api-high-cpu", resources[0].ID)
		assert.Equal(t, "api-high-cpu", resources[0].Name)

		body := requestBody(t, httpContext, 0)
		assert.Contains(t, body, "Action=DescribeAlarms")
		assert.Contains(t, body, "AlarmTypes.member.1=MetricAlarm")
	})
}

func Test__ListNamespaces(t *testing.T) {
	t.Run("returns distinct namespaces", func(t *testing.T) {
		ctx, httpContext := listResourcesContext(listMetricsXML)

		resources, err := ListNamespaces(ctx, "cloudwatch.namespace")
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "AWS/EC2", resources[0].ID)
		assert.Equal(t, "AWS/Lambda", resources[1].ID)

		assert.Contains(t, requestBody(t, httpContext, 0), "Action=ListMetrics")
	})
}

func Test__ListMetricNames(t *testing.T) {
	t.Run("missing namespace -> error", func(t *testing.T) {
		ctx, _ := listResourcesContext(listMetricsXML)

		_, err := ListMetricNames(ctx, "cloudwatch.metric")
		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("returns distinct metric names for the namespace", func(t *testing.T) {
		ctx, httpContext := listResourcesContext(listEC2CPUMetricsXML)
		ctx.Parameters["namespace"] = "AWS/EC2"

		resources, err := ListMetricNames(ctx, "cloudwatch.metric")
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "CPUUtilization", resources[0].ID)
		assert.Equal(t, "NetworkIn", resources[1].ID)

		assert.Contains(t, requestBody(t, httpContext, 0), "Namespace=AWS%2FEC2")
	})
}

func Test__ListMetricDimensions(t *testing.T) {
	t.Run("missing metric name -> error", func(t *testing.T) {
		ctx, _ := listResourcesContext(listMetricsXML)
		ctx.Parameters["namespace"] = "AWS/EC2"

		_, err := ListMetricDimensions(ctx, "cloudwatch.metricDimensions")
		require.ErrorContains(t, err, "metric name is required")
	})

	t.Run("returns each dimension set once, JSON-encoded", func(t *testing.T) {
		ctx, httpContext := listResourcesContext(listEC2CPUMetricsXML)
		ctx.Parameters["namespace"] = "AWS/EC2"
		ctx.Parameters["metricName"] = "CPUUtilization"

		resources, err := ListMetricDimensions(ctx, "cloudwatch.metricDimensions")
		require.NoError(t, err)
		require.Len(t, resources, 3)
		assert.Equal(t, "AutoScalingGroupName=api-asg", resources[0].Name)
		assert.Equal(t, `[{"name":"AutoScalingGroupName","value":"api-asg"}]`, resources[0].ID)
		assert.Equal(t, "InstanceId=i-1234567890abcdef0", resources[1].Name)
		assert.Equal(t, `[{"name":"InstanceId","value":"i-1234567890abcdef0"}]`, resources[1].ID)
		assert.Equal(t, "InstanceId=i-abcdef01234567890", resources[2].Name)

		body := requestBody(t, httpContext, 0)
		assert.Contains(t, body, "MetricName=CPUUtilization")
	})
}

func Test__Dimensions(t *testing.T) {
	t.Run("round-trips values containing separators", func(t *testing.T) {
		dimensions := []Dimension{{Name: "Path", Value: "a=b,c"}}

		encoded, err := EncodeDimensions(dimensions)
		require.NoError(t, err)

		decoded, err := DecodeDimensions(encoded)
		require.NoError(t, err)
		assert.Equal(t, dimensions, decoded)
	})

	t.Run("empty value decodes to no dimensions", func(t *testing.T) {
		decoded, err := DecodeDimensions("  ")
		require.NoError(t, err)
		assert.Empty(t, decoded)
	})
}
