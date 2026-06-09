package loop

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestLoopSetup(t *testing.T) {
	component := &Loop{}

	t.Run("requires until expression", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{}})
		require.Error(t, err)
		assert.ErrorContains(t, err, "untilExpression is required")
	})

	t.Run("accepts valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"untilExpression": `$["Checker"].ready == true`,
				"maxIterations":   10,
			},
		})
		require.NoError(t, err)
	})
}

func TestLoopStartLoop(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}
	execMetadata := &contexts.MetadataContext{}
	rootEventID := uuid.New().String()
	executionID := uuid.New()

	ctx := core.ProcessQueueContext{
		RootEventID: rootEventID,
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
		},
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return nil, nil
		},
		CreateExecution: func() (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				ID:             executionID,
				Metadata:       execMetadata,
				ExecutionState: execState,
			}, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, executionID, *id)
	assert.Equal(t, rootEventID, execState.KVs[loopSessionKey])
	md, ok := execMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, 1, md.Iteration)
	assert.True(t, md.Active)
	assert.True(t, execState.Passed)
	assert.Equal(t, ChannelNameBody, execState.Channel)
}

func TestLoopHandleFeedbackCompletesWhenUntilIsTrue(t *testing.T) {
	component := &Loop{}
	anchorMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Iteration: 2, Active: true},
	}
	iterationExecState := &contexts.ExecutionStateContext{}
	iterationID := uuid.New()
	anchorID := uuid.New()

	anchor := &core.ExecutionContext{
		ID:             anchorID,
		Metadata:       anchorMetadata,
		ExecutionState: &finishedExecutionState{},
	}

	ctx := core.ProcessQueueContext{
		RootEventID: uuid.New().String(),
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
		},
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return anchor, nil
		},
		Expressions: &contexts.ExpressionContext{Output: true},
		CreateExecution: func() (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				ID:             iterationID,
				ExecutionState: iterationExecState,
			}, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, iterationID, *id)
	assert.Equal(t, ChannelNameDone, iterationExecState.Channel)
	assert.Equal(t, PayloadTypeDone, iterationExecState.Type)

	donePayload := iterationExecState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, 2, donePayload["iteration"])
	assert.Equal(t, true, donePayload["completed"])
	md, ok := anchorMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.False(t, md.Active)
}

func TestLoopHandleFeedbackRepeatsBodyWhenUntilIsFalse(t *testing.T) {
	component := &Loop{}
	anchorMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Iteration: 1, Active: true},
	}
	iterationExecState := &contexts.ExecutionStateContext{}
	iterationID := uuid.New()

	anchor := &core.ExecutionContext{
		ID:             uuid.New(),
		Metadata:       anchorMetadata,
		ExecutionState: &finishedExecutionState{},
	}

	ctx := core.ProcessQueueContext{
		RootEventID: uuid.New().String(),
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
			"maxIterations":   5,
		},
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return anchor, nil
		},
		Expressions: &contexts.ExpressionContext{Output: false},
		CreateExecution: func() (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				ID:             iterationID,
				ExecutionState: iterationExecState,
			}, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, ChannelNameBody, iterationExecState.Channel)
	md, ok := anchorMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, 2, md.Iteration)
}

func TestLoopHandleFeedbackFailsAtMaxIterations(t *testing.T) {
	component := &Loop{}
	anchorMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Iteration: 3, Active: true},
	}
	anchorExecState := &contexts.ExecutionStateContext{Finished: true}

	anchor := &core.ExecutionContext{
		ID:             uuid.New(),
		Metadata:       anchorMetadata,
		ExecutionState: anchorExecState,
	}

	ctx := core.ProcessQueueContext{
		RootEventID: uuid.New().String(),
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
			"maxIterations":   3,
		},
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return anchor, nil
		},
		Expressions:     &contexts.ExpressionContext{Output: false},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.True(t, anchorExecState.Finished)
	assert.False(t, anchorExecState.Passed)
	md, ok := anchorMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.False(t, md.Active)
}

type finishedExecutionState struct {
	contexts.ExecutionStateContext
}

func (f *finishedExecutionState) IsFinished() bool {
	return true
}

func (f *finishedExecutionState) SetKV(key, value string) error { return nil }
func (f *finishedExecutionState) GetKV(key string) (string, error) {
	return "", core.ErrExecutionKVNotFound
}
func (f *finishedExecutionState) Emit(channel, payloadType string, payloads []any) error {
	return f.ExecutionStateContext.Emit(channel, payloadType, payloads)
}
func (f *finishedExecutionState) Pass() error { return f.ExecutionStateContext.Pass() }
func (f *finishedExecutionState) Fail(reason, message string) error {
	return f.ExecutionStateContext.Fail(reason, message)
}

func TestLoopExecuteIsNoOp(t *testing.T) {
	component := &Loop{}
	err := component.Execute(core.ExecutionContext{})
	require.NoError(t, err)
}

func TestLoopHandleWebhook(t *testing.T) {
	component := &Loop{}
	status, body, err := component.HandleWebhook(core.WebhookRequestContext{})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Nil(t, body)
}
