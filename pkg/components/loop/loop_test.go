package loop

import (
	"net/http"
	"testing"
	"time"

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

	t.Run("accepts delay between iterations configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"untilExpression": `$["Checker"].ready == true`,
				"delayBetweenIterations": map[string]any{
					"enabled":         true,
					"strategy":        DelayStrategyExponential,
					"intervalSeconds": 10,
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("accepts one second delay interval", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"untilExpression": `$["Checker"].ready == true`,
				"delayBetweenIterations": map[string]any{
					"enabled":         true,
					"strategy":        DelayStrategyFixed,
					"intervalSeconds": 1,
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("rejects invalid delay strategy", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"untilExpression": `$["Checker"].ready == true`,
				"delayBetweenIterations": map[string]any{
					"enabled":         true,
					"strategy":        "random",
					"intervalSeconds": 10,
				},
			},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid delay strategy")
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
	assert.Equal(t, ChannelNameNext, execState.Channel)
}

func TestReadMetadataFromPersistedJSON(t *testing.T) {
	startedAt := time.Now().Add(-3 * time.Second).UTC().Format(time.RFC3339Nano)
	md, err := readMetadata(&core.ExecutionContext{
		Metadata: &contexts.MetadataContext{
			Metadata: map[string]any{
				"iteration": float64(2),
				"active":    true,
				"startedAt": startedAt,
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, md.Iteration)
	assert.True(t, md.Active)
	assert.False(t, md.StartedAt.IsZero())
}

func TestLoopHandleFeedbackCompletesWhenUntilIsTrue(t *testing.T) {
	component := &Loop{}
	anchorMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{
			Iteration: 2,
			Active:    true,
			StartedAt: time.Now().Add(-2 * time.Second),
		},
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
	assert.Equal(t, 2, donePayload["iterations"])
	assert.Equal(t, StopReasonConditionMet, donePayload["stopReason"])
	elapsedMs, ok := donePayload["elapsedMs"].(int64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, elapsedMs, int64(2000))
	md, ok := anchorMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.False(t, md.Active)
}

func TestLoopHandleFeedbackRepeatsNextWhenUntilIsFalse(t *testing.T) {
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
	assert.Equal(t, ChannelNameNext, iterationExecState.Channel)
	md, ok := anchorMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, 2, md.Iteration)
}

func TestLoopHandleFeedbackSchedulesDelayBeforeNextIteration(t *testing.T) {
	component := &Loop{}
	anchorMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Iteration: 1, Active: true},
	}
	iterationMetadata := &contexts.MetadataContext{}
	scheduled := &scheduledRequestContext{}
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
			"delayBetweenIterations": map[string]any{
				"enabled":         true,
				"strategy":        DelayStrategyFixed,
				"intervalSeconds": 15,
			},
		},
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return anchor, nil
		},
		Expressions: &contexts.ExpressionContext{Output: false},
		CreateExecution: func() (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				ID:       iterationID,
				Metadata: iterationMetadata,
				Requests: scheduled,
			}, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.True(t, scheduled.called)
	assert.Equal(t, nextIterationHook, scheduled.actionName)
	assert.Equal(t, 15*time.Second, scheduled.interval)

	iterMd, ok := iterationMetadata.Metadata.(IterationExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, 2, iterMd.Iteration)
	assert.Equal(t, 5, iterMd.MaxIterations)
}

func TestLoopHandleNextIterationHook(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: nextIterationHook,
		Metadata: &contexts.MetadataContext{
			Metadata: IterationExecutionMetadata{
				Iteration:     3,
				MaxIterations: 10,
			},
		},
		ExecutionState: execState,
	})
	require.NoError(t, err)
	assert.Equal(t, ChannelNameNext, execState.Channel)
	assert.Equal(t, PayloadTypeNext, execState.Type)

	payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, 3, payload["iteration"])
	assert.Equal(t, 10, payload["maxIterations"])
}

func TestIterationDelay(t *testing.T) {
	delay := &DelaySpec{
		Enabled:         true,
		Strategy:        DelayStrategyFixed,
		IntervalSeconds: 15,
	}

	assert.Equal(t, time.Duration(0), iterationDelay(delay, 1))
	assert.Equal(t, 15*time.Second, iterationDelay(delay, 2))
	assert.Equal(t, 15*time.Second, iterationDelay(delay, 4))

	delay.Strategy = DelayStrategyExponential
	assert.Equal(t, 15*time.Second, iterationDelay(delay, 2))
	assert.Equal(t, 30*time.Second, iterationDelay(delay, 3))
	assert.Equal(t, 60*time.Second, iterationDelay(delay, 4))
}

func TestLoopHandleFeedbackCompletesAtMaxIterations(t *testing.T) {
	component := &Loop{}
	anchorMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{
			Iteration: 3,
			Active:    true,
			StartedAt: time.Now().Add(-1500 * time.Millisecond),
		},
	}
	doneExecState := &contexts.ExecutionStateContext{}
	doneExecutionID := uuid.New()

	anchor := &core.ExecutionContext{
		ID:             uuid.New(),
		Metadata:       anchorMetadata,
		ExecutionState: &finishedExecutionState{},
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
		Expressions: &contexts.ExpressionContext{Output: false},
		CreateExecution: func() (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				ID:             doneExecutionID,
				ExecutionState: doneExecState,
			}, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, doneExecutionID, *id)
	assert.Equal(t, ChannelNameDone, doneExecState.Channel)

	payload := doneExecState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, 3, payload["iterations"])
	assert.Equal(t, StopReasonMaxIterations, payload["stopReason"])
	elapsedMs, ok := payload["elapsedMs"].(int64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, elapsedMs, int64(1500))

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

type scheduledRequestContext struct {
	called     bool
	actionName string
	interval   time.Duration
}

func (s *scheduledRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	s.called = true
	s.actionName = actionName
	s.interval = interval
	return nil
}

func TestLoopExampleOutput(t *testing.T) {
	component := &Loop{}
	output := component.ExampleOutput()

	next, ok := output["next"].(map[string]any)
	require.True(t, ok)
	nextData, ok := next["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, PayloadTypeNext, next["type"])
	assert.Equal(t, 1, nextData["iteration"])

	done, ok := output["done"].(map[string]any)
	require.True(t, ok)
	doneData, ok := done["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, PayloadTypeDone, done["type"])
	assert.Equal(t, 3, doneData["iterations"])
	assert.Equal(t, StopReasonConditionMet, doneData["stopReason"])
	assert.Equal(t, int64(4521), doneData["elapsedMs"])
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
