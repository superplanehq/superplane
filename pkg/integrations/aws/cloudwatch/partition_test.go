package cloudwatch

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__PartitionForRegion(t *testing.T) {
	assert.Equal(t, "aws", PartitionForRegion("us-east-1"))
	assert.Equal(t, "aws", PartitionForRegion("eu-north-1"))
	assert.Equal(t, "aws-cn", PartitionForRegion("cn-north-1"))
	assert.Equal(t, "aws-cn", PartitionForRegion("cn-northwest-1"))
	assert.Equal(t, "aws-us-gov", PartitionForRegion("us-gov-west-1"))
}

func Test__APIDNSSuffix(t *testing.T) {
	assert.Equal(t, "amazonaws.com", APIDNSSuffix("us-east-1"))
	assert.Equal(t, "amazonaws.com", APIDNSSuffix("eu-north-1"))
	assert.Equal(t, "amazonaws.com.cn", APIDNSSuffix("cn-north-1"))
	assert.Equal(t, "amazonaws.com.cn", APIDNSSuffix("cn-northwest-1"))
	// GovCloud shares the default endpoint suffix.
	assert.Equal(t, "amazonaws.com", APIDNSSuffix("us-gov-west-1"))
}

func Test__ConsoleHost(t *testing.T) {
	assert.Equal(t, "console.aws.amazon.com", ConsoleHost("us-east-1"))
	assert.Equal(t, "console.amazonaws.cn", ConsoleHost("cn-north-1"))
}

func Test__ClientEndpoint(t *testing.T) {
	t.Run("standard regions", func(t *testing.T) {
		client := NewClient(nil, nil, "us-east-1")
		assert.Equal(t, "https://monitoring.us-east-1.amazonaws.com/", client.endpoint)
	})

	t.Run("china regions use the .cn suffix", func(t *testing.T) {
		client := NewClient(nil, nil, "cn-north-1")
		assert.Equal(t, "https://monitoring.cn-north-1.amazonaws.com.cn/", client.endpoint)
	})
}

func Test__LogsClientEndpoint(t *testing.T) {
	t.Run("standard regions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}},
		}

		client := NewLogsClient(httpContext, &aws.Credentials{AccessKeyID: "key", SecretAccessKey: "secret"}, "us-east-1")
		require.NoError(t, client.CreateLogStream("my-log-group", "my-stream"))

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://logs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})

	t.Run("china regions use the .cn suffix", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}},
		}

		client := NewLogsClient(httpContext, &aws.Credentials{AccessKeyID: "key", SecretAccessKey: "secret"}, "cn-north-1")
		require.NoError(t, client.CreateLogStream("my-log-group", "my-stream"))

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://logs.cn-north-1.amazonaws.com.cn/", httpContext.Requests[0].URL.String())
	})
}

func Test__AlarmConsoleURL(t *testing.T) {
	assert.Equal(t,
		"https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
		AlarmConsoleURL("us-east-1", "api-high-cpu"),
	)

	assert.Equal(t,
		"https://cn-north-1.console.amazonaws.cn/cloudwatch/home?region=cn-north-1#alarmsV2:alarm/api-high-cpu",
		AlarmConsoleURL("cn-north-1", "api-high-cpu"),
	)

	assert.Empty(t, AlarmConsoleURL("", "api-high-cpu"))
	assert.Empty(t, AlarmConsoleURL("us-east-1", ""))
}
