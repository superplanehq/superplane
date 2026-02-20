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
				"sourceKey": "source.txt",
				"bucket":    "dest-bucket",
				"key":       "dest.txt",
			},
		})
		require.ErrorContains(t, err, "bucket name is required")
	})

	t.Run("missing source key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "source-bucket",
				"bucket":       "dest-bucket",
				"key":          "dest.txt",
			},
		})
		require.ErrorContains(t, err, "object key is required")
	})

	t.Run("missing destination key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "source-bucket",
				"sourceKey":    "source.txt",
				"bucket":       "dest-bucket",
			},
		})
		require.ErrorContains(t, err, "object key is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "source-bucket",
				"sourceKey":    "source.txt",
				"bucket":       "dest-bucket",
				"key":          "dest.txt",
			},
		})
		require.NoError(t, err)
	})
}

func Test__CopyObject__Execute(t *testing.T) {
	component := &CopyObject{}

	t.Run("valid request -> emits copy object payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CopyObjectResult>
							<ETag>"abc123"</ETag>
							<LastModified>2026-01-10T10:00:02.000Z</LastModified>
						</CopyObjectResult>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"sourceBucket": "source-bucket",
				"sourceKey":    "source.txt",
				"bucket":       "dest-bucket",
				"key":          "dest.txt",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		result, ok := payload.(*CopyObjectResult)
		require.True(t, ok)
		assert.Equal(t, "source-bucket", result.SourceBucket)
		assert.Equal(t, "source.txt", result.SourceKey)
		assert.Equal(t, "dest-bucket", result.Bucket)
		assert.Equal(t, "dest.txt", result.Key)
		assert.Equal(t, "abc123", result.ETag)
		assert.Equal(t, "2026-01-10T10:00:02.000Z", result.LastModified)
	})
}
