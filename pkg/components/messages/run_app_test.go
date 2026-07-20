package messages

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunApp__FailureMessage(t *testing.T) {
	timedOutAt := "2026-07-20T12:00:00Z"
	assert.Equal(t, "timed out after 30s", runAppFailureMessage(runAppExecutionMetadata{
		TimedOutAt: &timedOutAt,
	}, map[string]any{"timeout": 30}, nil))
	assert.Equal(t, "timed out after 3600s", runAppFailureMessage(runAppExecutionMetadata{
		TimedOutAt: &timedOutAt,
	}, map[string]any{}, nil))

	runError := "child failed"
	assert.Equal(t, "child failed", runAppFailureMessage(runAppExecutionMetadata{}, nil, &runError))
	assert.Equal(t, "", runAppFailureMessage(runAppExecutionMetadata{}, nil, nil))
}

func Test__RunApp__TimeoutSeconds(t *testing.T) {
	defaultTimeout := runAppTimeoutSeconds(nil)
	assert.Equal(t, defaultRunAppTimeoutSeconds, defaultTimeout)

	zero := 0
	assert.Equal(t, defaultRunAppTimeoutSeconds, runAppTimeoutSeconds(&zero))

	custom := 120
	assert.Equal(t, 120, runAppTimeoutSeconds(&custom))
}

func Test__RunApp__Execute__SchedulesConfiguredTimeout(t *testing.T) {
	childAppID := uuid.New()
	runID := uuid.New()
	requests := &contexts.RequestContext{}
	executionMetadata := &contexts.MetadataContext{}
	nodeMetadata := &contexts.MetadataContext{
		Metadata: RunAppMetadata{
			App:  &AppMetadata{ID: childAppID.String(), Name: "Child App"},
			Node: &CanvasNodeMetadata{ID: "onRun", Name: "On Run"},
		},
	}
	runs := &contexts.RunExecutionContext{CreateRunID: runID}

	err := (&RunApp{}).Execute(core.ExecutionContext{
		WorkflowID: "parent-workflow",
		CanvasName: "Parent App",
		Configuration: map[string]any{
			"app":        childAppID.String(),
			"node":       "onRun",
			"parameters": map[string]any{},
			"timeout":    45,
		},
		Metadata:     executionMetadata,
		NodeMetadata: nodeMetadata,
		Requests:     requests,
		Runs:         runs,
	})
	require.NoError(t, err)

	assert.Equal(t, ActionRunTimeout, requests.Action)
	assert.Equal(t, 45*time.Second, requests.Duration)

	metadata := decodeRunAppExecutionMetadata(t, executionMetadata)
	require.NotNil(t, metadata.Run)
	assert.Equal(t, runID.String(), metadata.Run.ID)
	assert.Nil(t, metadata.TimedOutAt)

	require.NotNil(t, runs.LastCreateParams)
	assert.Equal(t, childAppID.String(), runs.LastCreateParams.App)
	assert.Equal(t, "onRun", runs.LastCreateParams.Node)
	require.Len(t, runs.LastCreateParams.Callbacks, 2)
}

func Test__RunApp__Execute__SchedulesDefaultTimeout(t *testing.T) {
	childAppID := uuid.New()
	requests := &contexts.RequestContext{}
	executionMetadata := &contexts.MetadataContext{}
	nodeMetadata := &contexts.MetadataContext{
		Metadata: RunAppMetadata{
			App:  &AppMetadata{ID: childAppID.String(), Name: "Child App"},
			Node: &CanvasNodeMetadata{ID: "onRun", Name: "On Run"},
		},
	}

	err := (&RunApp{}).Execute(core.ExecutionContext{
		WorkflowID: "parent-workflow",
		CanvasName: "Parent App",
		Configuration: map[string]any{
			"app":        childAppID.String(),
			"node":       "onRun",
			"parameters": map[string]any{},
		},
		Metadata:     executionMetadata,
		NodeMetadata: nodeMetadata,
		Requests:     requests,
		Runs:         &contexts.RunExecutionContext{},
	})
	require.NoError(t, err)

	assert.Equal(t, ActionRunTimeout, requests.Action)
	assert.Equal(t, time.Duration(defaultRunAppTimeoutSeconds)*time.Second, requests.Duration)
}

func Test__RunApp__HandleRunTimeout__CancelsChildRun(t *testing.T) {
	childRunID := uuid.New().String()
	metadataCtx := &contexts.MetadataContext{
		Metadata: runAppExecutionMetadata{
			Run: &RunMetadata{ID: childRunID},
		},
	}
	runs := &contexts.RunExecutionContext{}

	err := (&RunApp{}).handleRunTimeout(core.ActionHookContext{
		Metadata:       metadataCtx,
		ExecutionState: &contexts.ExecutionStateContext{},
		Runs:           runs,
	})
	require.NoError(t, err)
	assert.True(t, runs.CancelCalled)

	metadata := decodeRunAppExecutionMetadata(t, metadataCtx)
	require.NotNil(t, metadata.TimedOutAt)
	assert.NotEmpty(t, *metadata.TimedOutAt)
	assert.Equal(t, childRunID, metadata.Run.ID)
}

func Test__RunApp__HandleRunTimeout__NoOpWhenFinished(t *testing.T) {
	childRunID := uuid.New().String()
	metadataCtx := &contexts.MetadataContext{
		Metadata: runAppExecutionMetadata{
			Run: &RunMetadata{ID: childRunID},
		},
	}
	runs := &contexts.RunExecutionContext{}

	err := (&RunApp{}).handleRunTimeout(core.ActionHookContext{
		Metadata:       metadataCtx,
		ExecutionState: &contexts.ExecutionStateContext{Finished: true},
		Runs:           runs,
	})
	require.NoError(t, err)
	assert.False(t, runs.CancelCalled)

	metadata := decodeRunAppExecutionMetadata(t, metadataCtx)
	assert.Nil(t, metadata.TimedOutAt)
}

func Test__RunApp__HandleRunFinished__EmitsFailedWithTimeoutMessage(t *testing.T) {
	timedOutAt := time.Now().UTC().Format(time.RFC3339)
	childRunID := uuid.New()
	metadataCtx := &contexts.MetadataContext{
		Metadata: runAppExecutionMetadata{
			Run:        &RunMetadata{ID: childRunID.String()},
			TimedOutAt: &timedOutAt,
		},
	}
	execState := &contexts.ExecutionStateContext{}

	params, err := core.NewRunFinishedCallback(core.NewRun(
		childRunID,
		uuid.New(),
		core.RunResultCancelled,
		nil,
	)).ToParameters()
	require.NoError(t, err)

	err = (&RunApp{}).handleRunFinished(core.ActionHookContext{
		Metadata:       metadataCtx,
		ExecutionState: execState,
		Parameters:     params,
		Configuration: map[string]any{
			"timeout": 5,
		},
	})
	require.NoError(t, err)

	assert.Equal(t, FailedOutputChannel, execState.Channel)
	assert.True(t, execState.Finished)

	metadata := decodeRunAppExecutionMetadata(t, metadataCtx)
	require.NotNil(t, metadata.Run)
	require.NotNil(t, metadata.Run.Error)
	assert.Equal(t, "timed out after 5s", *metadata.Run.Error)
	assert.Equal(t, core.RunResultCancelled, metadata.Run.Result)
	assert.Equal(t, &timedOutAt, metadata.TimedOutAt)
}

func decodeRunAppExecutionMetadata(t *testing.T, metadataCtx *contexts.MetadataContext) runAppExecutionMetadata {
	t.Helper()

	var metadata runAppExecutionMetadata
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
	return metadata
}
