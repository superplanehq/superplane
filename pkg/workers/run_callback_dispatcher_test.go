package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__RunCallbackDispatcher__DispatchPending__NoCallback(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "entry", Type: models.NodeTypeTrigger}},
		nil,
	)

	run := createPendingRun(t, canvas.ID, "entry", nil)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return NewRunCallbackDispatcher(tx, r.Registry, run).DispatchPending()
	})
	require.NoError(t, err)
}

func Test__RunCallbackDispatcher__DispatchPending__OnEntry__EmitsRootEvent(t *testing.T) {
	triggerName := "run_cb_pending_entry_" + uuid.New().String()
	registry.RegisterTrigger(triggerName, impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name:  triggerName,
		Hooks: []core.Hook{{Name: "onMessage", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.TriggerHookContext) (map[string]any, error) {
			err := ctx.Events.Emit("test.invocation", map[string]any{
				"parameters": ctx.Parameters,
			})
			return nil, err
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID: "entry",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
			},
		},
		nil,
	)

	run := createPendingRun(t, canvas.ID, "entry", []core.RunCallback{
		{
			When: core.RunCallbackWhenPending,
			On:   core.RunCallbackOnEntry,
			Hook: "onMessage",
		},
	})
	require.NoError(t, database.Conn().Model(run).Update("input", models.NewJSONValue(map[string]any{
		"app":        map[string]any{"id": "parent-app", "name": "Parent"},
		"parameters": map[string]any{"key": "value"},
	})).Error)

	var collectedEvents []models.CanvasEvent
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return NewRunCallbackDispatcher(tx, r.Registry, run).
			WithEventCollector(func(events []models.CanvasEvent) {
				collectedEvents = append(collectedEvents, events...)
			}).
			DispatchPending()
	})
	require.NoError(t, err)
	require.Len(t, collectedEvents, 1)
	assert.Equal(t, run.ID, collectedEvents[0].RunID)
	assert.Equal(t, "entry", collectedEvents[0].NodeID)
}

func Test__RunCallbackDispatcher__DispatchPending__OnParent__InvokesParentHook(t *testing.T) {
	actionName := "run_cb_pending_parent_" + uuid.New().String()
	hookCalled := false
	var receivedParams map[string]any

	registry.RegisterAction(actionName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  actionName,
		Hooks: []core.Hook{{Name: "onMessage", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			receivedParams = ctx.Parameters
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: actionName}}),
			},
		},
		nil,
	)
	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "entry", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createStartedRunRecord(t, parentCanvas.ID, "trigger")
	parentExecution := createRunExecutionRecord(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID, models.CanvasNodeExecutionStateStarted, "")

	childRun := createSubRunWithCallbacks(t, subRunOptions{
		workflowID:        childCanvas.ID,
		nodeID:            "entry",
		parentRunID:       parentRun.ID,
		parentWorkflowID:  parentCanvas.ID,
		parentExecutionID: parentExecution.ID,
		state:             models.CanvasRunStatePending,
		callbacks: []core.RunCallback{
			{
				When: core.RunCallbackWhenPending,
				On:   core.RunCallbackOnParent,
				Hook: "onMessage",
			},
		},
		input: map[string]any{
			"app":        map[string]any{"id": parentCanvas.ID.String(), "name": "Parent"},
			"parameters": map[string]any{"env": "staging"},
		},
	})

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return NewRunCallbackDispatcher(tx, r.Registry, childRun).DispatchPending()
	})
	require.NoError(t, err)
	assert.True(t, hookCalled)
	assert.Equal(t, "staging", receivedParams["parameters"].(map[string]any)["env"])
}

func Test__RunCallbackDispatcher__DispatchFinished__NoCallback(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "entry", Type: models.NodeTypeTrigger}},
		nil,
	)

	run := createStartedRunRecord(t, canvas.ID, "entry")

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return NewRunCallbackDispatcher(tx, r.Registry, run).DispatchFinished()
	})
	require.NoError(t, err)
}

func Test__RunCallbackDispatcher__DispatchFinished__OnParent__InvokesParentHook(t *testing.T) {
	actionName := "run_cb_finished_parent_" + uuid.New().String()
	hookCalled := false
	var receivedCallback core.RunFinishedCallback

	registry.RegisterAction(actionName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  actionName,
		Hooks: []core.Hook{{Name: "onRunFinished", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			var err error
			receivedCallback, err = core.DecodeRunFinishedCallback(ctx.Parameters)
			return err
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: actionName}}),
			},
		},
		nil,
	)
	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "entry", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createStartedRunRecord(t, parentCanvas.ID, "trigger")
	parentExecution := createRunExecutionRecord(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID, models.CanvasNodeExecutionStateStarted, "")

	childRun := createSubRunWithCallbacks(t, subRunOptions{
		workflowID:        childCanvas.ID,
		nodeID:            "entry",
		parentRunID:       parentRun.ID,
		parentWorkflowID:  parentCanvas.ID,
		parentExecutionID: parentExecution.ID,
		state:             models.CanvasRunStateFinished,
		result:            models.CanvasRunResultPassed,
		callbacks: []core.RunCallback{
			{
				When: core.RunCallbackWhenFinished,
				On:   core.RunCallbackOnParent,
				Hook: "onRunFinished",
			},
		},
	})

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return NewRunCallbackDispatcher(tx, r.Registry, childRun).DispatchFinished()
	})
	require.NoError(t, err)
	assert.True(t, hookCalled)
	assert.Equal(t, childRun.ID, receivedCallback.Run.ID)
	assert.Equal(t, childCanvas.ID, receivedCallback.Run.AppID)
	assert.Equal(t, models.CanvasRunResultPassed, receivedCallback.Run.Result)
}

func Test__RunCallbackDispatcher__DispatchFinished__OnParent__SkipsWhenParentCancelling(t *testing.T) {
	actionName := "run_cb_finished_skip_cancel_" + uuid.New().String()
	hookCalled := false

	registry.RegisterAction(actionName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  actionName,
		Hooks: []core.Hook{{Name: "onRunFinished", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: actionName}}),
			},
		},
		nil,
	)
	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "entry", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createStartedRunRecord(t, parentCanvas.ID, "trigger")
	parentExecution := createRunExecutionRecord(
		t,
		parentCanvas.ID,
		parentRun.ID,
		parentNodes[1].NodeID,
		models.CanvasNodeExecutionStateCancelling,
		"",
	)

	childRun := createSubRunWithCallbacks(t, subRunOptions{
		workflowID:        childCanvas.ID,
		nodeID:            "entry",
		parentRunID:       parentRun.ID,
		parentWorkflowID:  parentCanvas.ID,
		parentExecutionID: parentExecution.ID,
		state:             models.CanvasRunStateFinished,
		result:            models.CanvasRunResultPassed,
		callbacks: []core.RunCallback{
			{
				When: core.RunCallbackWhenFinished,
				On:   core.RunCallbackOnParent,
				Hook: "onRunFinished",
			},
		},
	})

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return NewRunCallbackDispatcher(tx, r.Registry, childRun).DispatchFinished()
	})
	require.NoError(t, err)
	assert.False(t, hookCalled)
}

func Test__RunCallbackDispatcher__DispatchFinished__OnParent__SkipsWhenParentCancelled(t *testing.T) {
	actionName := "run_cb_finished_skip_cancelled_" + uuid.New().String()
	hookCalled := false

	registry.RegisterAction(actionName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  actionName,
		Hooks: []core.Hook{{Name: "onRunFinished", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: actionName}}),
			},
		},
		nil,
	)
	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "entry", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createStartedRunRecord(t, parentCanvas.ID, "trigger")
	parentExecution := createRunExecutionRecord(
		t,
		parentCanvas.ID,
		parentRun.ID,
		parentNodes[1].NodeID,
		models.CanvasNodeExecutionStateFinished,
		models.CanvasNodeExecutionResultCancelled,
	)

	childRun := createSubRunWithCallbacks(t, subRunOptions{
		workflowID:        childCanvas.ID,
		nodeID:            "entry",
		parentRunID:       parentRun.ID,
		parentWorkflowID:  parentCanvas.ID,
		parentExecutionID: parentExecution.ID,
		state:             models.CanvasRunStateFinished,
		result:            models.CanvasRunResultPassed,
		callbacks: []core.RunCallback{
			{
				When: core.RunCallbackWhenFinished,
				On:   core.RunCallbackOnParent,
				Hook: "onRunFinished",
			},
		},
	})

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return NewRunCallbackDispatcher(tx, r.Registry, childRun).DispatchFinished()
	})
	require.NoError(t, err)
	assert.False(t, hookCalled)
}

type subRunOptions struct {
	workflowID        uuid.UUID
	nodeID            string
	parentRunID       uuid.UUID
	parentWorkflowID  uuid.UUID
	parentExecutionID uuid.UUID
	state             string
	result            string
	callbacks         []core.RunCallback
	input             map[string]any
}

func createSubRunWithCallbacks(t *testing.T, opts subRunOptions) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), opts.workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:                uuid.New(),
		WorkflowID:        opts.workflowID,
		NodeID:            opts.nodeID,
		VersionID:         liveVersion.ID,
		ParentRunID:       &opts.parentRunID,
		ParentWorkflowID:  &opts.parentWorkflowID,
		ParentExecutionID: &opts.parentExecutionID,
		Callbacks:         datatypes.NewJSONSlice(opts.callbacks),
		State:             opts.state,
		Result:            opts.result,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	if opts.input != nil {
		run.Input = models.NewJSONValue(opts.input)
	}

	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func createStartedRunRecord(t *testing.T, workflowID uuid.UUID, nodeID string) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		VersionID:  liveVersion.ID,
		State:      models.CanvasRunStateStarted,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func createRunExecutionRecord(
	t *testing.T,
	workflowID uuid.UUID,
	runID uuid.UUID,
	nodeID string,
	state string,
	result string,
) *models.CanvasNodeExecution {
	t.Helper()

	rootEvent := support.EmitCanvasEventForNode(t, workflowID, nodeID, "default", nil)
	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		RootEventID:   rootEvent.ID,
		RunID:         runID,
		EventID:       rootEvent.ID,
		State:         state,
		Result:        result,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}
