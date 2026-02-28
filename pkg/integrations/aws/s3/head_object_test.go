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

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"key":    "file.txt",
			},
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
		})
		require.ErrorContains(t, err, "key is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
				"key":    "file.txt",
			},
		})
		require.NoError(t, err)
	})
}

func Test__HeadObject__Execute(t *testing.T) {
	component := &HeadObject{}

	t.Run("valid request -> emits object metadata payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Length":      []string{"1048576"},
						"Content-Type":        []string{"application/pdf"},
						"Etag":                []string{"\"d41d8cd98f00b204e9800998ecf8427e\""},
						"Last-Modified":       []string{"Fri, 28 Feb 2026 10:00:00 GMT"},
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
				"key":    "documents/report.pdf",
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
		metadata, ok := payload.(*ObjectMetadata)
		require.True(t, ok)
		assert.Equal(t, "documents/report.pdf", metadata.Key)
		assert.Equal(t, "my-bucket", metadata.Bucket)
		assert.Equal(t, int64(1048576), metadata.ContentLength)
		assert.Equal(t, "application/pdf", metadata.ContentType)
		assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", metadata.ETag)
		assert.Equal(t, "STANDARD", metadata.StorageClass)
	})
}
