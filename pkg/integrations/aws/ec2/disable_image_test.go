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

func Test__DisableImage__Execute(t *testing.T) {
	component := &DisableImage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DisableImageResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-disable</requestId>
						<return>true</return>
					</DisableImageResponse>
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
	assert.Equal(t, "aws.ec2.image.disabled", execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"]
	output, ok := payload.(map[string]any)
	require.True(t, ok)
	image, ok := output["image"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ami-123", image["imageId"])
}
