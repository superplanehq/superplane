package railway

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

func Test__Railway__GetDeployment__Setup(t *testing.T) {
	action := &GetDeployment{}

	require.ErrorContains(t, action.Setup(core.SetupContext{Configuration: map[string]any{}}), "deployId is required")
	require.NoError(t, action.Setup(core.SetupContext{Configuration: map[string]any{"deployId": "deploy-123"}}))
}

func Test__Railway__GetDeployment__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"deployment":{"id":"deploy-123","status":"SUCCESS","createdAt":"2026-05-30T00:00:00Z","updatedAt":"2026-05-30T00:01:00Z","projectId":"p-1","serviceId":"s-1","environmentId":"e-1","canRollback":true,"canRedeploy":true,"deploymentStopped":false}}}`)),
			},
		},
	}
	intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
	_ = intCtx.SetSecret("apiToken", []byte("test-token"))
	stateCtx := &contexts.ExecutionStateContext{}

	err := (&GetDeployment{}).Execute(core.ExecutionContext{
		HTTP:           httpCtx,
		Integration:    intCtx,
		Configuration:  map[string]any{"deployId": "deploy-123"},
		ExecutionState: stateCtx,
	})
	require.NoError(t, err)

	assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
	assert.Equal(t, GetDeploymentPayloadType, stateCtx.Type)
	require.Len(t, stateCtx.Payloads, 1)
	payload := stateCtx.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "deploy-123", payload["deployId"])
	assert.Equal(t, "SUCCESS", payload["status"])
	assert.Equal(t, true, payload["canRollback"])
}
