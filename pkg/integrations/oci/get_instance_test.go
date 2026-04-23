package oci

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetInstance__Setup(t *testing.T) {
	component := &GetInstance{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{"instanceId": ""},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "instanceId is required")
}

func Test__GetInstance__Execute(t *testing.T) {
	component := &GetInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateRunning)),
			ociMockResponse(http.StatusOK, `[{"vnicId":"ocid1.vnic.oc1.test","lifecycleState":"ATTACHED"}]`),
			ociMockResponse(http.StatusOK, `{"id":"ocid1.vnic.oc1.test","publicIp":"192.0.2.1","privateIp":"10.0.0.17"}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"instanceId": testInstanceID},
		HTTP:           httpCtx,
		Integration:    ociIntegrationContext(),
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 3)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/20160918/instances/"+testInstanceID)
	assert.Contains(t, httpCtx.Requests[1].URL.String(), "/20160918/vnicAttachments")
	assert.Contains(t, httpCtx.Requests[2].URL.String(), "/20160918/vnics/ocid1.vnic.oc1.test")

	assert.True(t, executionState.Passed)
	assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	assert.Equal(t, GetInstancePayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	wrapped := executionState.Payloads[0].(map[string]any)
	data := wrapped["data"].(map[string]any)
	assert.Equal(t, testInstanceID, data["instanceId"])
	assert.Equal(t, "192.0.2.1", data["publicIp"])
	assert.Equal(t, "10.0.0.17", data["privateIp"])
}
