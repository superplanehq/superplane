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

	t.Run("rejects negative timeout", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"untilExpression": `$["Checker"].ready == true`,
				"timeoutSeconds":  -5,
			},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "timeoutSeconds must be at least")
	})

	t.Run("rejects timeout above maximum", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"untilExpression": `$["Checker"].ready == true`,
				"timeoutSeconds":  TimeoutMaxSeconds + 1,
			},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "timeoutSeconds cannot exceed")
	})

	t.Run("rejects maxIterations above maximum", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"untilExpression": `$["Checker"].ready == true`,
				"maxIterations":   MaxIterationsLimit + 1,
			},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "maxIterations cannot exceed")
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
	scheduled := &scheduledRequestContext{}
	rootEventID := uuid.New().String()
	executionID := uuid.New()

	ctx := core.ProcessQueueContext{
		RootEventID: rootEventID,
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
			"timeoutSeconds":  120,
		},
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return nil, nil
		},
		CreateExecution: func() (*core.ExecutionContext, error) {
			return &core.ExecutionContext{
				ID:             executionID,
				Metadata:       execMetadata,
				Requests:       scheduled,
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
	assert.False(t, execState.Finished)
	assert.Equal(t, ChannelNameNext, execState.Channel)
	assert.True(t, scheduled.called)
	assert.Equal(t, timeoutHook, scheduled.actionName)
	assert.Equal(t, 120*time.Second, scheduled.interval)
}

func TestLoopStartDeferredWhenAnotherSessionActive(t *testing.T) {
	component := &Loop{}
	deferred := false

	ctx := core.ProcessQueueContext{
		RootEventID: uuid.New().String(),
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
		},
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			return nil, nil
		},
		HasRunningExecutions: func() (bool, error) {
			return true, nil
		},
		DeferQueueItem: func() error {
			deferred = true
			return nil
		},
		CreateExecution: func() (*core.ExecutionContext, error) {
			t.Fatal("CreateExecution should not be called when deferring")
			return nil, nil
		},
	}

	id, err := component.ProcessQueueItem(ctx)
	require.ErrorIs(t, err, core.ErrQueueItemDeferred)
	assert.Nil(t, id)
	assert.True(t, deferred)
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
			Iteration:     2,
			MaxIterations: 100,
			Active:        true,
			StartedAt:     time.Now().Add(-2 * time.Second),
		},
	}
	anchorExecState := &contexts.ExecutionStateContext{}
	anchorID := uuid.New()

	anchor := &core.ExecutionContext{
		ID:             anchorID,
		Metadata:       anchorMetadata,
		ExecutionState: anchorExecState,
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
			t.Fatal("feedback must reuse the session execution")
			return nil, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, anchorID, *id)
	assert.Equal(t, ChannelNameDone, anchorExecState.Channel)
	assert.Equal(t, PayloadTypeDone, anchorExecState.Type)
	assert.True(t, anchorExecState.Finished)

	donePayload := anchorExecState.Payloads[0].(map[string]any)["data"].(map[string]any)["done"].(map[string]any)
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
		Metadata: ExecutionMetadata{Iteration: 1, MaxIterations: 5, Active: true},
	}
	anchorExecState := &contexts.ExecutionStateContext{}
	anchorID := uuid.New()

	anchor := &core.ExecutionContext{
		ID:             anchorID,
		Metadata:       anchorMetadata,
		ExecutionState: anchorExecState,
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
			t.Fatal("feedback must reuse the session execution")
			return nil, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, anchorID, *id)
	assert.Equal(t, ChannelNameNext, anchorExecState.Channel)
	assert.False(t, anchorExecState.Finished)
	md, ok := anchorMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, 2, md.Iteration)
}

func TestLoopHandleFeedbackSchedulesDelayBeforeNextIteration(t *testing.T) {
	component := &Loop{}
	anchorMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Iteration: 1, MaxIterations: 5, Active: true},
	}
	scheduled := &scheduledRequestContext{}
	anchorID := uuid.New()

	anchor := &core.ExecutionContext{
		ID:             anchorID,
		Metadata:       anchorMetadata,
		Requests:       scheduled,
		ExecutionState: &contexts.ExecutionStateContext{},
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
			t.Fatal("feedback must reuse the session execution")
			return nil, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, anchorID, *id)
	assert.True(t, scheduled.called)
	assert.Equal(t, nextIterationHook, scheduled.actionName)
	assert.Equal(t, 15*time.Second, scheduled.interval)

	md, ok := anchorMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, 2, md.Iteration)
	assert.True(t, md.WaitingBetweenIterations)
}

func TestLoopHandleFeedbackIgnoresStaleFeedbackWhenFinished(t *testing.T) {
	component := &Loop{}
	anchor := &core.ExecutionContext{
		ID:             uuid.New(),
		Metadata:       &contexts.MetadataContext{},
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
		CreateExecution: func() (*core.ExecutionContext, error) {
			t.Fatal("stale feedback must not create a new execution")
			return nil, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	assert.Nil(t, id)
}

func TestLoopHandleNextIterationHook(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: nextIterationHook,
		Metadata: &contexts.MetadataContext{
			Metadata: ExecutionMetadata{
				Iteration:     3,
				MaxIterations: 10,
				Active:        true,
			},
		},
		ExecutionState: execState,
	})
	require.NoError(t, err)
	assert.Equal(t, ChannelNameNext, execState.Channel)
	assert.Equal(t, PayloadTypeNext, execState.Type)
	assert.False(t, execState.Finished)

	payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)["next"].(map[string]any)
	assert.Equal(t, 3, payload["iteration"])
	assert.Equal(t, 10, payload["maxIterations"])
}

func TestLoopHandleTimeoutHookFailsActiveLoop(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}
	execMetadata := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Iteration: 4, MaxIterations: 10, Active: true},
	}

	err := component.HandleHook(core.ActionHookContext{
		Name: timeoutHook,
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
			"timeoutSeconds":  30,
		},
		Metadata:       execMetadata,
		ExecutionState: execState,
	})
	require.NoError(t, err)
	assert.True(t, execState.Finished)
	assert.False(t, execState.Passed)
	assert.Equal(t, "timeout", execState.FailureReason)
	assert.Contains(t, execState.FailureMessage, "30s")

	md, ok := execMetadata.Metadata.(ExecutionMetadata)
	require.True(t, ok)
	assert.False(t, md.Active)
}

func TestLoopHandleTimeoutHookIgnoresFinishedLoop(t *testing.T) {
	component := &Loop{}

	err := component.HandleHook(core.ActionHookContext{
		Name: timeoutHook,
		Configuration: map[string]any{
			"untilExpression": `$["Checker"].ready == true`,
		},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &finishedExecutionState{},
	})
	require.NoError(t, err)
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
			Iteration:     3,
			MaxIterations: 3,
			Active:        true,
			StartedAt:     time.Now().Add(-1500 * time.Millisecond),
		},
	}
	anchorExecState := &contexts.ExecutionStateContext{}
	anchorID := uuid.New()

	anchor := &core.ExecutionContext{
		ID:             anchorID,
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
		Expressions: &contexts.ExpressionContext{Output: false},
		CreateExecution: func() (*core.ExecutionContext, error) {
			t.Fatal("feedback must reuse the session execution")
			return nil, nil
		},
		DequeueItem:     func() error { return nil },
		UpdateNodeState: func(state string) error { return nil },
	}

	id, err := component.ProcessQueueItem(ctx)
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, anchorID, *id)
	assert.Equal(t, ChannelNameDone, anchorExecState.Channel)

	payload := anchorExecState.Payloads[0].(map[string]any)["data"].(map[string]any)["done"].(map[string]any)
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
func (f *finishedExecutionState) EmitAndContinue(channel, payloadType string, payloads []any) error {
	return f.ExecutionStateContext.EmitAndContinue(channel, payloadType, payloads)
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

	// The example output is a single payload in the standard envelope format
	assert.Equal(t, PayloadTypeDone, output["type"])
	assert.NotEmpty(t, output["timestamp"])

	data, ok := output["data"].(map[string]any)
	require.True(t, ok)

	doneData, ok := data["done"].(map[string]any)
	require.True(t, ok)
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
