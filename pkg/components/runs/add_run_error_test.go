package runs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestAddRunError_Execute_RecordsViaRunsContext(t *testing.T) {
	runsCtx := &contexts.RunExecutionContext{}
	stateCtx := &contexts.ExecutionStateContext{}

	component := &AddRunError{}
	err := component.Execute(core.ExecutionContext{
		NodeID:   "deploy",
		NodeName: "Deploy",
		Configuration: map[string]any{
			"message": "pipeline failed",
		},
		Runs:           runsCtx,
		ExecutionState: stateCtx,
	})

	require.NoError(t, err)
	require.Len(t, runsCtx.AddErrorCalls, 1)
	assert.Equal(t, "pipeline failed", runsCtx.AddErrorCalls[0])
	assert.True(t, stateCtx.Passed)
	assert.Equal(t, addRunErrorPayloadType, stateCtx.Type)
}

func TestAddRunError_Execute_RequiresMessage(t *testing.T) {
	component := &AddRunError{}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{},
		Runs:           &contexts.RunExecutionContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
	})

	require.ErrorContains(t, err, "message is required")
}

func TestAddRunError_Execute_FailsExecutionWhenErrorsTooLarge(t *testing.T) {
	runsCtx := &contexts.RunExecutionContext{
		AddErrorErr: models.ErrRunErrorsTooLarge,
	}
	stateCtx := &contexts.ExecutionStateContext{}

	component := &AddRunError{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"message": "pipeline failed",
		},
		Runs:           runsCtx,
		ExecutionState: stateCtx,
	})

	require.NoError(t, err)
	assert.False(t, stateCtx.Passed)
	assert.Equal(t, models.CanvasNodeExecutionResultReasonError, stateCtx.FailureReason)
	assert.Contains(t, stateCtx.FailureMessage, "run errors exceed maximum size")
}
