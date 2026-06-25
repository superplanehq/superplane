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

func Test__GetBucket__Setup(t *testing.T) {
	component := &GetBucket{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": " ", "bucket": "my-bucket"},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-east-1", "bucket": " "},
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-east-1", "bucket": "my-bucket"},
		})
		require.NoError(t, err)
	})
}

func Test__GetBucket__Execute(t *testing.T) {
	component := &GetBucket{}

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "us-east-1", "bucket": "my-bucket"},
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits resolved region", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">eu-west-1</LocationConstraint>`,
				)),
			}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "eu-west-1", "bucket": "my-bucket"},
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
		assert.Equal(t, "eu-west-1", data["region"])
		assert.Equal(t, "arn:aws:s3:::my-bucket", data["arn"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "location")
	})

	t.Run("empty location constraint -> defaults to us-east-1", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`,
				)),
			}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "us-east-1", "bucket": "my-bucket"},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: awsSecrets()},
		})

		require.NoError(t, err)
		data := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "us-east-1", data["region"])
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`<Error><Code>NoSuchBucket</Code></Error>`)),
			}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "us-east-1", "bucket": "my-bucket"},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: awsSecrets()},
		})

		require.ErrorContains(t, err, "failed to get S3 bucket")
		assert.False(t, execState.Passed)
	})
}
