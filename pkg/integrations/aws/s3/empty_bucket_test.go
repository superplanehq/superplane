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

func Test__EmptyBucket__Setup(t *testing.T) {
	component := &EmptyBucket{}

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
		})
		require.NoError(t, err)
	})
}

func Test__EmptyBucket__Execute(t *testing.T) {
	component := &EmptyBucket{}

	t.Run("bucket with objects -> lists and deletes all objects", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<ListBucketResult>
							<IsTruncated>false</IsTruncated>
							<Contents>
								<Key>file1.txt</Key>
								<Size>100</Size>
							</Contents>
							<Contents>
								<Key>file2.txt</Key>
								<Size>200</Size>
							</Contents>
						</ListBucketResult>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DeleteResult>
							<Deleted><Key>file1.txt</Key></Deleted>
							<Deleted><Key>file2.txt</Key></Deleted>
						</DeleteResult>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<ListBucketResult>
							<IsTruncated>false</IsTruncated>
						</ListBucketResult>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		result, ok := payload.(*EmptyBucketResult)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", result.Bucket)
		assert.Equal(t, 2, result.DeletedCount)
	})

	t.Run("empty bucket -> emits result with zero deleted count", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<ListBucketResult>
							<IsTruncated>false</IsTruncated>
						</ListBucketResult>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "empty-bucket",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		result, ok := payload.(*EmptyBucketResult)
		require.True(t, ok)
		assert.Equal(t, "empty-bucket", result.Bucket)
		assert.Equal(t, 0, result.DeletedCount)
	})
}
