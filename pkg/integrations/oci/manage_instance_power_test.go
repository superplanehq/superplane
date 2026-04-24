package oci

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ManageInstancePower__Setup(t *testing.T) {
	component := &ManageInstancePower{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{"instanceId": "", "action": "STOP"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instanceId is required")

	err = component.Setup(core.SetupContext{
		Configuration: map[string]any{"instanceId": testInstanceID, "action": ""},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action is required")
}

func Test__ManageInstancePower__Execute(t *testing.T) {
	component := &ManageInstancePower{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, ociInstanceBody("STOPPING")),
		},
	}
	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{"instanceId": testInstanceID, "action": "STOP"},
		HTTP:          httpCtx,
		Integration:   ociIntegrationContext(),
		Metadata:      metadata,
		Requests:      requests,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	assert.Equal(t, "STOP", httpCtx.Requests[0].URL.Query().Get("action"))
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, ManageInstancePowerMetadata{
		InstanceID:  testInstanceID,
		Action:      "STOP",
		TargetState: instanceStateStopped,
	}, metadata.Metadata)
}

func Test__ManageInstancePower__PollReschedulesUntilTargetState(t *testing.T) {
	component := &ManageInstancePower{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateRunning)),
		},
	}
	metadata := &contexts.MetadataContext{
		Metadata: ManageInstancePowerMetadata{
			InstanceID:  testInstanceID,
			Action:      "STOP",
			TargetState: instanceStateStopped,
		},
	}
	requests := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleAction(core.ActionContext{
		Name:           "poll",
		HTTP:           httpCtx,
		Integration:    ociIntegrationContext(),
		Metadata:       metadata,
		Requests:       requests,
		ExecutionState: executionState,
		Logger:         ociLogger(),
	})

	require.NoError(t, err)
	assert.False(t, executionState.Passed)
	assert.Equal(t, "poll", requests.Action)
}

func Test__ManageInstancePower__PollEmitsAtTargetState(t *testing.T) {
	component := &ManageInstancePower{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateStopped)),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleAction(core.ActionContext{
		Name:        "poll",
		HTTP:        httpCtx,
		Integration: ociIntegrationContext(),
		Metadata: &contexts.MetadataContext{
			Metadata: ManageInstancePowerMetadata{
				InstanceID:  testInstanceID,
				Action:      "STOP",
				TargetState: instanceStateStopped,
			},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: executionState,
		Logger:         ociLogger(),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, ManageInstancePowerPayloadType, executionState.Type)
}

func Test__ManageInstancePower__PollHandlesConsecutiveErrors(t *testing.T) {
	component := &ManageInstancePower{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusInternalServerError, `{"code":"InternalError"}`),
		},
	}
	metadata := &contexts.MetadataContext{
		Metadata: ManageInstancePowerMetadata{
			InstanceID:  testInstanceID,
			Action:      "STOP",
			TargetState: instanceStateStopped,
			PollErrors:  maxPollErrors - 1,
		},
	}

	err := component.HandleAction(core.ActionContext{
		Name:           "poll",
		HTTP:           httpCtx,
		Integration:    ociIntegrationContext(),
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         ociLogger(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "giving up polling instance")
}
