package oci

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateInstance__Setup(t *testing.T) {
	component := &UpdateInstance{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{"instanceId": ""},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "instanceId is required")
}

func Test__UpdateInstance__Execute(t *testing.T) {
	component := &UpdateInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateRunning)),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"instanceId":  testInstanceID,
			"displayName": "renamed",
			"ocpus":       2.0,
			"memoryInGBs": 16.0,
		},
		HTTP:           httpCtx,
		Integration:    ociIntegrationContext(),
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	req := httpCtx.Requests[0]
	assert.Equal(t, http.MethodPut, req.Method)
	assert.Contains(t, req.URL.String(), "/20160918/instances/"+testInstanceID)

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "renamed", payload["displayName"])
	require.Contains(t, payload, "shapeConfig")
	shapeConfig := payload["shapeConfig"].(map[string]any)
	assert.Equal(t, float64(2), shapeConfig["ocpus"])
	assert.Equal(t, float64(16), shapeConfig["memoryInGBs"])

	assert.True(t, executionState.Passed)
	assert.Equal(t, UpdateInstancePayloadType, executionState.Type)
}
