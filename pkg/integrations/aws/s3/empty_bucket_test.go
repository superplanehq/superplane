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

func Test__EmptyBucket__Setup(t *testing.T) {
	component := &EmptyBucket{}

	t.Run("missing bucket -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "bucket name is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
		})
		require.NoError(t, err)
	})
}

func Test__EmptyBucket__Execute(t *testing.T) {
	component := &EmptyBucket{}

	t.Run("bucket already empty -> emits emptied payload with zero count", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<ListBucketResult>
							<IsTruncated>false</IsTruncated>
						</ListBucketResult>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-bucket", payload["bucket"])
		assert.Equal(t, 0, payload["objectsCount"])
	})

	t.Run("bucket with objects -> lists and deletes, then emits", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<ListBucketResult>
							<Contents>
								<Key>file1.txt</Key>
								<ETag>"etag1"</ETag>
								<Size>100</Size>
							</Contents>
							<Contents>
								<Key>file2.txt</Key>
								<ETag>"etag2"</ETag>
								<Size>200</Size>
							</Contents>
							<IsTruncated>false</IsTruncated>
						</ListBucketResult>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DeleteResult></DeleteResult>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<ListBucketResult>
							<IsTruncated>false</IsTruncated>
						</ListBucketResult>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"bucket": "my-bucket",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    validIntegrationContext(),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-bucket", payload["bucket"])
		assert.Equal(t, 2, payload["objectsCount"])
	})
}
