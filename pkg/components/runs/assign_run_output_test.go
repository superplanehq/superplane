package runs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestAssignRunOutput_Execute_MergesViaRunsContext(t *testing.T) {
	runsCtx := &contexts.RunExecutionContext{}
	stateCtx := &contexts.ExecutionStateContext{}

	component := &AssignRunOutput{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"output": map[string]any{
				"deploy": map[string]any{"id": "d-1"},
			},
		},
		Runs:           runsCtx,
		ExecutionState: stateCtx,
	})

	require.NoError(t, err)
	require.Len(t, runsCtx.AssignOutputCalls, 1)
	assert.Equal(t, map[string]any{
		"deploy": map[string]any{"id": "d-1"},
	}, runsCtx.AssignOutputCalls[0])
	assert.True(t, stateCtx.Passed)
	assert.Equal(t, assignRunOutputPayloadType, stateCtx.Type)
}

func TestAssignRunOutput_Execute_RequiresOutputObject(t *testing.T) {
	component := &AssignRunOutput{}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{},
		Runs:           &contexts.RunExecutionContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
	})

	require.ErrorContains(t, err, "output is required")
}

func TestAssignRunOutput_Execute_FailsExecutionWhenOutputTooLarge(t *testing.T) {
	runsCtx := &contexts.RunExecutionContext{
		AssignOutputErr: models.ErrRunOutputTooLarge,
	}
	stateCtx := &contexts.ExecutionStateContext{}

	component := &AssignRunOutput{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"output": map[string]any{"key": "value"},
		},
		Runs:           runsCtx,
		ExecutionState: stateCtx,
	})

	require.NoError(t, err)
	assert.False(t, stateCtx.Passed)
	assert.Equal(t, models.CanvasNodeExecutionResultReasonError, stateCtx.FailureReason)
	assert.Contains(t, stateCtx.FailureMessage, "run output exceeds maximum size")
}
