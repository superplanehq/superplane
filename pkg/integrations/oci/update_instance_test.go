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
		Configuration: map[string]any{"instance": ""},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "instance is required")
}

func Test__UpdateInstance__Execute(t *testing.T) {
	component := &UpdateInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateRunning)),
			ociMockResponse(http.StatusOK, `[{"vnicId":"ocid1.vnic.oc1.test","lifecycleState":"ATTACHED"}]`),
			ociMockResponse(http.StatusOK, `{"id":"ocid1.vnic.oc1.test","publicIp":"192.0.2.1","privateIp":"10.0.0.17"}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"instance":    testInstanceID,
			"displayName": "renamed",
			"ocpus":       2.0,
			"memoryInGBs": 16.0,
		},
		HTTP:           httpCtx,
		Integration:    ociIntegrationContext(),
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 3)
	req := httpCtx.Requests[0]
	assert.Equal(t, http.MethodPut, req.Method)
	assert.Contains(t, req.URL.String(), "/20160918/instances/"+testInstanceID)
	assert.Contains(t, httpCtx.Requests[1].URL.String(), "/20160918/vnicAttachments")
	assert.Contains(t, httpCtx.Requests[2].URL.String(), "/20160918/vnics/ocid1.vnic.oc1.test")

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
	require.Len(t, executionState.Payloads, 1)

	wrapped := executionState.Payloads[0].(map[string]any)
	data := wrapped["data"].(map[string]any)
	assert.Equal(t, "192.0.2.1", data["publicIp"])
	assert.Equal(t, "10.0.0.17", data["privateIp"])
}
