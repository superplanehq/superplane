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

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"key":     "file.txt",
				"content": "hello",
			},
		})
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"bucket":  "my-bucket",
				"content": "hello",
			},
		})
		require.ErrorContains(t, err, "key is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"bucket":  "my-bucket",
				"key":     "file.txt",
				"content": "hello world",
			},
		})
		require.NoError(t, err)
	})
}

func Test__PutObject__Execute(t *testing.T) {
	component := &PutObject{}

	t.Run("valid request -> emits put object result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Etag": []string{"\"d41d8cd98f00b204e9800998ecf8427e\""},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		contentType := "application/json"
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":      "us-east-1",
				"bucket":      "my-bucket",
				"key":         "data/output.json",
				"content":     `{"status": "ok"}`,
				"contentType": contentType,
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
		result, ok := payload.(*PutObjectResult)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", result.Bucket)
		assert.Equal(t, "data/output.json", result.Key)
		assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", result.ETag)
	})
}
