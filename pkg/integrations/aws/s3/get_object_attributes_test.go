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

func Test__GetObjectAttributes__Setup(t *testing.T) {
	component := &GetObjectAttributes{}

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

func Test__GetObjectAttributes__Execute(t *testing.T) {
	component := &GetObjectAttributes{}

	t.Run("valid request -> emits object attributes payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<GetObjectAttributesResponse>
							<ETag>d41d8cd98f00b204e9800998ecf8427e</ETag>
							<StorageClass>STANDARD</StorageClass>
							<ObjectSize>1048576</ObjectSize>
						</GetObjectAttributesResponse>
					`)),
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
		attrs, ok := payload.(*ObjectAttributes)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", attrs.Bucket)
		assert.Equal(t, "documents/report.pdf", attrs.Key)
		assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", attrs.ETag)
		assert.Equal(t, "STANDARD", attrs.StorageClass)
		assert.Equal(t, int64(1048576), attrs.ObjectSize)
	})
}
