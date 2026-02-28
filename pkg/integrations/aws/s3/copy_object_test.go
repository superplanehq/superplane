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

func Test__CopyObject__Setup(t *testing.T) {
	component := &CopyObject{}

	t.Run("missing source bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"sourceKey": "src/file.txt",
				"bucket":    "dest-bucket",
				"key":       "dest/file.txt",
			},
		})
		require.ErrorContains(t, err, "source bucket is required")
	})

	t.Run("missing source key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "src-bucket",
				"bucket":       "dest-bucket",
				"key":          "dest/file.txt",
			},
		})
		require.ErrorContains(t, err, "source key is required")
	})

	t.Run("missing destination bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "src-bucket",
				"sourceKey":    "src/file.txt",
				"key":          "dest/file.txt",
			},
		})
		require.ErrorContains(t, err, "destination bucket is required")
	})

	t.Run("missing destination key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "src-bucket",
				"sourceKey":    "src/file.txt",
				"bucket":       "dest-bucket",
			},
		})
		require.ErrorContains(t, err, "destination key is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "src-bucket",
				"sourceKey":    "src/file.txt",
				"bucket":       "dest-bucket",
				"key":          "dest/file.txt",
			},
		})
		require.NoError(t, err)
	})
}

func Test__CopyObject__Execute(t *testing.T) {
	component := &CopyObject{}

	t.Run("valid request -> emits copy result payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CopyObjectResult>
							<ETag>"d41d8cd98f00b204e9800998ecf8427e"</ETag>
							<LastModified>2026-02-28T10:00:00.000Z</LastModified>
						</CopyObjectResult>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "src-bucket",
				"sourceKey":    "src/file.txt",
				"bucket":       "dest-bucket",
				"key":          "dest/file.txt",
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
		result, ok := payload.(*CopyObjectResult)
		require.True(t, ok)
		assert.Equal(t, "src-bucket", result.SourceBucket)
		assert.Equal(t, "src/file.txt", result.SourceKey)
		assert.Equal(t, "dest-bucket", result.Bucket)
		assert.Equal(t, "dest/file.txt", result.Key)
		assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", result.ETag)
	})
}
