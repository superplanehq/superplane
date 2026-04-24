package oci

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteInstance__Setup(t *testing.T) {
	component := &DeleteInstance{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{"instanceId": ""},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "instanceId is required")
}

func Test__DeleteInstance__Execute(t *testing.T) {
	component := &DeleteInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusNoContent, ``),
		},
	}
	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{"instanceId": testInstanceID, "preserveBootVolume": true},
		HTTP:          httpCtx,
		Integration:   ociIntegrationContext(),
		Metadata:      metadata,
		Requests:      requests,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
	assert.Equal(t, "true", httpCtx.Requests[0].URL.Query().Get("preserveBootVolume"))
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, DeleteInstanceMetadata{InstanceID: testInstanceID}, metadata.Metadata)
}

func Test__DeleteInstance__PollEmitsWhenTerminated(t *testing.T) {
	component := &DeleteInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateTerminated)),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleAction(core.ActionContext{
		Name:        "poll",
		HTTP:        httpCtx,
		Integration: ociIntegrationContext(),
		Metadata: &contexts.MetadataContext{
			Metadata: DeleteInstanceMetadata{InstanceID: testInstanceID},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: executionState,
		Logger:         ociLogger(),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, DeleteInstancePayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)
	wrapped := executionState.Payloads[0].(map[string]any)
	data := wrapped["data"].(map[string]any)
	assert.Equal(t, instanceStateTerminated, data["lifecycleState"])
}

func Test__DeleteInstance__PollTreats404AsTerminated(t *testing.T) {
	component := &DeleteInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusNotFound, `{"code":"NotAuthorizedOrNotFound"}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleAction(core.ActionContext{
		Name:        "poll",
		HTTP:        httpCtx,
		Integration: ociIntegrationContext(),
		Metadata: &contexts.MetadataContext{
			Metadata: DeleteInstanceMetadata{InstanceID: testInstanceID},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: executionState,
		Logger:         ociLogger(),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, DeleteInstancePayloadType, executionState.Type)
}

func Test__DeleteInstance__PollHandlesTransientErrors(t *testing.T) {
	component := &DeleteInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusInternalServerError, `{"code":"InternalError"}`),
		},
	}
	metadata := &contexts.MetadataContext{
		Metadata: DeleteInstanceMetadata{InstanceID: testInstanceID},
	}
	requests := &contexts.RequestContext{}

	err := component.HandleAction(core.ActionContext{
		Name:           "poll",
		HTTP:           httpCtx,
		Integration:    ociIntegrationContext(),
		Metadata:       metadata,
		Requests:       requests,
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         ociLogger(),
	})

	require.NoError(t, err)
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, DeleteInstanceMetadata{InstanceID: testInstanceID, PollErrors: 1}, metadata.Metadata)
}

func Test__DeleteInstance__PollStopsAfterMaxErrors(t *testing.T) {
	component := &DeleteInstance{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			ociMockResponse(http.StatusInternalServerError, `{"code":"InternalError"}`),
		},
	}

	err := component.HandleAction(core.ActionContext{
		Name:        "poll",
		HTTP:        httpCtx,
		Integration: ociIntegrationContext(),
		Metadata: &contexts.MetadataContext{
			Metadata: DeleteInstanceMetadata{InstanceID: testInstanceID, PollErrors: maxPollErrors - 1},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         ociLogger(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "giving up polling instance")
}
