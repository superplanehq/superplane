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

func Test__PutObject__Setup(t *testing.T) {
	component := &PutObject{}

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
		})
		require.ErrorContains(t, err, "object key is required")
	})

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"key":    "path/to/object.txt",
			},
		})
		require.ErrorContains(t, err, "bucket name is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
				"key":    "path/to/object.txt",
				"body":   "hello world",
			},
		})
		require.NoError(t, err)
	})
}

func Test__PutObject__Execute(t *testing.T) {
	component := &PutObject{}

	t.Run("valid request -> emits put object payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Etag": []string{"\"abc123\""},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"bucket":      "my-bucket",
				"key":         "path/to/object.txt",
				"body":        "hello world",
				"contentType": "text/plain",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		result, ok := payload.(*PutObjectResult)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", result.Bucket)
		assert.Equal(t, "path/to/object.txt", result.Key)
		assert.Equal(t, "abc123", result.ETag)
	})
}
