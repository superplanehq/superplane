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
				"region":            " ",
				"sourceBucket":      "src",
				"sourceKey":         "key",
				"destinationBucket": "dst",
				"destinationKey":    "key",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing source bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"sourceBucket":      " ",
				"sourceKey":         "key",
				"destinationBucket": "dst",
				"destinationKey":    "key",
			},
		})

		require.ErrorContains(t, err, "source bucket is required")
	})

	t.Run("missing source key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"sourceBucket":      "src",
				"sourceKey":         " ",
				"destinationBucket": "dst",
				"destinationKey":    "key",
			},
		})

		require.ErrorContains(t, err, "source key is required")
	})

	t.Run("missing destination bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"sourceBucket":      "src",
				"sourceKey":         "key",
				"destinationBucket": " ",
				"destinationKey":    "key",
			},
		})

		require.ErrorContains(t, err, "destination bucket is required")
	})

	t.Run("missing destination key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"sourceBucket":      "src",
				"sourceKey":         "key",
				"destinationBucket": "dst",
				"destinationKey":    " ",
			},
		})

		require.ErrorContains(t, err, "destination key is required")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"sourceBucket":      "src-bucket",
				"sourceKey":         "source/file.txt",
				"destinationBucket": "dst-bucket",
				"destinationKey":    "dest/file.txt",
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
				"region":            "us-east-1",
				"sourceBucket":      "src",
				"sourceKey":         "key",
				"destinationBucket": "dst",
				"destinationKey":    "key",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("valid request -> emits copy result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CopyObjectResult>
							<ETag>"d41d8cd98f00b204e9800998ecf8427e"</ETag>
							<LastModified>2026-02-24T12:00:00.000Z</LastModified>
						</CopyObjectResult>
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":            "us-east-1",
				"sourceBucket":      "src-bucket",
				"sourceKey":         "source/file.txt",
				"destinationBucket": "dst-bucket",
				"destinationKey":    "dest/file.txt",
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
		assert.Equal(t, "src-bucket", data["sourceBucket"])
		assert.Equal(t, "source/file.txt", data["sourceKey"])
		assert.Equal(t, "dst-bucket", data["destinationBucket"])
		assert.Equal(t, "dest/file.txt", data["destinationKey"])

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "https://s3.us-east-1.amazonaws.com/dst-bucket/")
		assert.Equal(t, "/src-bucket/source%2Ffile.txt", httpContext.Requests[0].Header.Get("x-amz-copy-source"))
	})
}
