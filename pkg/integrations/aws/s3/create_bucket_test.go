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

func Test__CreateBucket__Setup(t *testing.T) {
	component := &CreateBucket{}

	t.Run("missing name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
		})
		require.ErrorContains(t, err, "bucket name is required")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "my-bucket",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"name":   "my-bucket",
			},
		})
		require.NoError(t, err)
	})
}

func Test__CreateBucket__Execute(t *testing.T) {
	component := &CreateBucket{}

	t.Run("valid request -> emits created bucket payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Location": []string{"/my-bucket"},
					},
					Body: io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1",
				"name":   "my-bucket",
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
		bucket, ok := payload.(*Bucket)
		require.True(t, ok)
		assert.Equal(t, "my-bucket", bucket.Name)
		assert.Equal(t, "/my-bucket", bucket.Location)
	})
}
