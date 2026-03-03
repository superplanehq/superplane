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

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": " ",
				"bucket": "my-bucket",
				"key":    "file.txt",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": " ",
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
				"key":    " ",
			},
		})

		require.ErrorContains(t, err, "key is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
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

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
				"key":    "file.txt",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits object metadata", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		headers.Set("Content-Length", "1024")
		headers.Set("ETag", `"d41d8cd98f00b204e9800998ecf8427e"`)
		headers.Set("Last-Modified", "Tue, 11 Feb 2026 12:00:00 GMT")
		headers.Set("X-Amz-Storage-Class", "STANDARD")

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header:     headers,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
				"key":    "path/to/file.txt",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "aws.s3.object", execState.Type)

		require.Len(t, execState.Payloads, 1)
		payloadWrapper, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := payloadWrapper["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", data["bucket"])
		assert.Equal(t, "path/to/file.txt", data["key"])
		assert.Equal(t, "application/json", data["contentType"])
		assert.Equal(t, "1024", data["contentLength"])
	})
}
