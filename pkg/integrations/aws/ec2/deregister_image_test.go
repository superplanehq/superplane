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

func Test__DeregisterImage__Execute(t *testing.T) {
	component := &DeregisterImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DeregisterImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-deregister</requestId>
						<return>true</return>
					</DeregisterImageResponse>
				`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"region":  "us-east-1",
			"imageId": "ami-123",
		},
		HTTP:           httpContext,
		ExecutionState: execState,
		Integration:    testIntegrationWithCredentials(),
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 1)
	assert.Equal(t, "aws.ec2.image.deregistered", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(DeregisterImageOutput)
	require.True(t, ok)
	assert.Equal(t, "ami-123", output.ImageID)
	assert.True(t, output.Deregistered)

	require.Len(t, httpContext.Requests, 1)
	requestBody := testRequestBodyString(t, httpContext.Requests[0])
	assert.Contains(t, requestBody, "Action=DeregisterImage")
	assert.Contains(t, requestBody, "ImageId=ami-123")
}
