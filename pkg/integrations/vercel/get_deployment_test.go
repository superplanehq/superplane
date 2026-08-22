package vercel

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

func Test__Vercel_GetDeployment__Setup(t *testing.T) {
	t.Run("missing deploymentId -> error", func(t *testing.T) {
		err := (&GetDeployment{}).Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "deploymentId is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := (&GetDeployment{}).Setup(core.SetupContext{
			Configuration: map[string]any{"deploymentId": "dpl_1"},
		})
		require.NoError(t, err)
	})
}

func Test__Vercel_GetDeployment__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"id":"dpl_1","url":"my-app.vercel.app","readyState":"READY","target":"production","projectId":"prj_1","createdAt":1755850000000}`,
				)),
			},
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	err := (&GetDeployment{}).Execute(core.ExecutionContext{
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
		ExecutionState: executionState,
		Configuration:  map[string]any{"deploymentId": "dpl_1"},
	})

	require.NoError(t, err)

	assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	assert.Equal(t, GetDeploymentPayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	emittedPayload := readMap(executionState.Payloads[0])
	data := readMap(emittedPayload["data"])
	assert.Equal(t, "dpl_1", data["deploymentId"])
	assert.Equal(t, "READY", data["readyState"])
	assert.Equal(t, "prj_1", data["projectId"])

	require.Len(t, httpCtx.Requests, 1)
	request := httpCtx.Requests[0]
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, "/v13/deployments/dpl_1", request.URL.Path)

	integrationRequest := httpCtx.Requests[0]
	assert.Equal(t, "Bearer vercel_token_123", integrationRequest.Header.Get("Authorization"))
}
