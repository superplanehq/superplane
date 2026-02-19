package ec2

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

func Test__EnableImageDeprecation__Setup(t *testing.T) {
	component := &EnableImageDeprecation{}

	t.Run("deprecateAt is required -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":  "us-east-1",
				"imageId": "ami-123",
			},
		})

		require.ErrorContains(t, err, "deprecateAt is required")
	})
}

func Test__EnableImageDeprecation__Execute(t *testing.T) {
	component := &EnableImageDeprecation{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<EnableImageDeprecationResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-enable-deprecation</requestId>
						<return>true</return>
					</EnableImageDeprecationResponse>
				`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":      "us-east-1",
			"imageId":     "ami-123",
			"deprecateAt": "2026-03-01T00:00:00Z",
		},
		HTTP:           httpContext,
		ExecutionState: execState,
		Integration:    testIntegrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.deprecationEnabled", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(map[string]any)
	require.True(t, ok)
	image, ok := output["image"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ami-123", image["imageId"])
	assert.Equal(t, "2026-03-01T00:00:00Z", output["deprecateAt"])
	assert.Equal(t, "req-enable-deprecation", output["requestId"])

	require.Len(t, httpContext.Requests, 1)
	requestBody := testRequestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, requestBody, "Action=EnableImageDeprecation")
	assert.Contains(t, requestBody, "ImageId=ami-123")
	assert.Contains(t, requestBody, "DeprecateAt=2026-03-01T00%3A00%3A00Z")
}
