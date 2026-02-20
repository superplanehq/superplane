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

	t.Run("missing attributes -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "my-bucket",
				"key":        "path/to/object.txt",
				"attributes": []string{},
			},
		})
		require.ErrorContains(t, err, "at least one attribute is required")
	})

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "my-bucket",
				"attributes": []string{"ETag"},
			},
		})
		require.ErrorContains(t, err, "object key is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "my-bucket",
				"key":        "path/to/object.txt",
				"attributes": []string{"ETag", "ObjectSize"},
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
							<ETag>abc123</ETag>
							<StorageClass>STANDARD</StorageClass>
							<ObjectSize>2048</ObjectSize>
						</GetObjectAttributesResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":     "us-east-1",
				"bucket":     "my-bucket",
				"key":        "path/to/object.txt",
				"attributes": []string{"ETag", "StorageClass", "ObjectSize"},
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"]
		result, ok := payload.(*ObjectAttributes)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", result.Bucket)
		assert.Equal(t, "path/to/object.txt", result.Key)
		assert.Equal(t, "abc123", result.ETag)
		assert.Equal(t, "STANDARD", result.StorageClass)
		assert.Equal(t, int64(2048), result.ObjectSize)
	})
}
