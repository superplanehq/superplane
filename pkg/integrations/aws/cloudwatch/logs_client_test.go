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

func TestLogsClient_Endpoint(t *testing.T) {
	t.Run("default partition -> amazonaws.com", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}},
		}

		client := NewLogsClient(httpContext, &aws.Credentials{AccessKeyID: "key", SecretAccessKey: "secret"}, "us-east-1")
		require.NoError(t, client.CreateLogStream("my-log-group", "my-stream"))

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://logs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})

	t.Run("China partition -> amazonaws.com.cn", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}},
		}

		client := NewLogsClient(httpContext, &aws.Credentials{AccessKeyID: "key", SecretAccessKey: "secret"}, "cn-north-1")
		require.NoError(t, client.CreateLogStream("my-log-group", "my-stream"))

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://logs.cn-north-1.amazonaws.com.cn/", httpContext.Requests[0].URL.String())
	})
}
