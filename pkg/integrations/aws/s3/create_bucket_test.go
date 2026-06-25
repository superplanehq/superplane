package s3

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

func awsSecrets() map[string]core.IntegrationSecret {
	return map[string]core.IntegrationSecret{
		"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
		"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
		"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
	}
}

func Test__CreateBucket__Setup(t *testing.T) {
	component := &CreateBucket{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": " ", "bucketName": "my-bucket"},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing bucket name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-east-1", "bucketName": " "},
		})
		require.ErrorContains(t, err, "bucket name is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-east-1", "bucketName": "my-bucket"},
		})
		require.NoError(t, err)
	})
}

func Test__CreateBucket__Execute(t *testing.T) {
	component := &CreateBucket{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "us-east-1", "bucketName": "my-bucket"},
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("us-east-1 -> creates bucket without location constraint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: http.NoBody}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "us-east-1", "bucketName": "my-bucket"},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: awsSecrets()},
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "aws.s3.bucket", execState.Type)

		require.Len(t, execState.Payloads, 1)
		data := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-bucket", data["bucketName"])
		assert.Equal(t, "us-east-1", data["region"])
		assert.Equal(t, "arn:aws:s3:::my-bucket", data["arn"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Equal(t, "https://s3.us-east-1.amazonaws.com/my-bucket", httpContext.Requests[0].URL.String())
	})

	t.Run("non us-east-1 -> sends location constraint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: http.NoBody}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "eu-west-1", "bucketName": "my-bucket"},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: awsSecrets()},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		bodyBytes, _ := io.ReadAll(httpContext.Requests[0].Body)
		assert.Contains(t, string(bodyBytes), "<LocationConstraint>eu-west-1</LocationConstraint>")
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusConflict,
				Body:       io.NopCloser(strings.NewReader(`<Error><Code>BucketAlreadyExists</Code></Error>`)),
			}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "us-east-1", "bucketName": "my-bucket"},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: awsSecrets()},
		})

		require.ErrorContains(t, err, "failed to create S3 bucket")
		assert.False(t, execState.Passed)
	})
}
