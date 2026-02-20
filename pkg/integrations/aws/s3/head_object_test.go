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

func Test__HeadObject__Setup(t *testing.T) {
	component := &HeadObject{}

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
		})
		require.ErrorContains(t, err, "object key is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
				"key":    "path/to/object.txt",
			},
		})
		require.NoError(t, err)
	})
}

func Test__HeadObject__Execute(t *testing.T) {
	component := &HeadObject{}

	t.Run("valid request -> emits head object payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Length":      []string{"1024"},
						"Content-Type":        []string{"text/plain"},
						"Etag":                []string{"\"abc123\""},
						"Last-Modified":       []string{"Thu, 10 Jan 2026 10:00:02 GMT"},
						"X-Amz-Storage-Class": []string{"STANDARD"},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
				"key":    "path/to/object.txt",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		result, ok := payload.(*HeadObjectResult)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", result.Bucket)
		assert.Equal(t, "path/to/object.txt", result.Key)
		assert.Equal(t, "1024", result.ContentLength)
		assert.Equal(t, "text/plain", result.ContentType)
		assert.Equal(t, "abc123", result.ETag)
		assert.Equal(t, "STANDARD", result.StorageClass)
	})
}
