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

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     " ",
				"bucket":     "dest-bucket",
				"key":        "dest/file.txt",
				"copySource": "src-bucket/src/file.txt",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     " ",
				"key":        "dest/file.txt",
				"copySource": "src-bucket/src/file.txt",
			},
		})

		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "dest-bucket",
				"key":        " ",
				"copySource": "src-bucket/src/file.txt",
			},
		})

		require.ErrorContains(t, err, "key is required")
	})

	t.Run("missing copy source -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "dest-bucket",
				"key":        "dest/file.txt",
				"copySource": " ",
			},
		})

		require.ErrorContains(t, err, "copy source is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "dest-bucket",
				"key":        "dest/file.txt",
				"copySource": "src-bucket/src/file.txt",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CopyObject__Execute(t *testing.T) {
	component := &CopyObject{}

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
				"region":     "us-east-1",
				"bucket":     "dest-bucket",
				"key":        "dest/file.txt",
				"copySource": "src-bucket/src/file.txt",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits copy details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CopyObjectResult>
						  <ETag>"d41d8cd98f00b204e9800998ecf8427e"</ETag>
						  <LastModified>2026-02-11T12:00:00.000Z</LastModified>
						</CopyObjectResult>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "dest-bucket",
				"key":        "dest/file.txt",
				"copySource": "src-bucket/src/file.txt",
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
		assert.Equal(t, "aws.s3.object.copied", execState.Type)

		require.Len(t, execState.Payloads, 1)
		payloadWrapper, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := payloadWrapper["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "dest-bucket", data["bucket"])
		assert.Equal(t, "dest/file.txt", data["key"])
		assert.Equal(t, "src-bucket/src/file.txt", data["copySource"])
	})
}
