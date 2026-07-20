package models_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__CanvasRun__DeleteChain__DeletesAllData(t *testing.T) {
	run, execution := setupRunWithExecution(t)

	childEvent := support.EmitCanvasEventForNode(t, run.WorkflowID, "node-1", "default", &execution.ID)
	require.NoError(t, database.Conn().Model(childEvent).Update("state", models.CanvasEventStateRouted).Error)

	queueItem := support.CreateQueueItem(t, run.WorkflowID, "node-1", execution.RootEventID, execution.RootEventID)
	queueItem.RunID = run.ID
	require.NoError(t, database.Conn().Save(queueItem).Error)

	require.NoError(t, models.CreateNodeExecutionKVInTransaction(database.Conn(), run.WorkflowID, "node-1", execution.ID, "test-key", "test-value"))

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  run.WorkflowID,
		NodeID:      "node-1",
		ExecutionID: &execution.ID,
		State:       models.NodeExecutionRequestStateCompleted,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{ActionName: "test", Parameters: map[string]any{}},
		}),
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	var summary *models.RunDeletionSummary
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		summary, err = run.DeleteChain(tx)
		return err
	}))

	require.Equal(t, &models.RunDeletionSummary{
		Runs:             1,
		Events:           2,
		NodeExecutions:   1,
		NodeRequests:     1,
		NodeExecutionKVs: 1,
		NodeQueueItems:   1,
	}, summary)

	var runCount int64
	require.NoError(t, database.Conn().Model(&models.CanvasRun{}).Where("id = ?", run.ID).Count(&runCount).Error)
	require.Equal(t, int64(0), runCount)

	support.VerifyCanvasEventsCount(t, run.WorkflowID, 0)
	support.VerifyNodeExecutionsCount(t, run.WorkflowID, 0)
	support.VerifyNodeQueueCount(t, run.WorkflowID, 0)
	support.VerifyNodeExecutionKVCount(t, run.WorkflowID, 0)
	support.VerifyNodeRequestCount(t, run.WorkflowID, 0)
}

func Test__CanvasRun__FindOpenWork__PendingOutputEvent(t *testing.T) {
	run, execution := setupRunWithExecution(t)

	_, err := execution.Pass(map[string][]any{
		"default": {map[string]any{"data": "output"}},
	})
	require.NoError(t, err)

	openWork, err := run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.True(t, openWork.HasPendingEvents)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasQueueItems)

	var outputEvent models.CanvasEvent
	require.NoError(t, database.Conn().
		Where("execution_id = ?", execution.ID).
		First(&outputEvent).
		Error)
	assert.Equal(t, run.ID, outputEvent.RunID)

	require.NoError(t, outputEvent.Routed())
	openWork, err = run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.False(t, openWork.HasPendingEvents)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasQueueItems)
}

func Test__CanvasRun__FindOpenWork__QueueItem(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	queueItem := support.CreateQueueItem(t, run.WorkflowID, "node-1", execution.RootEventID, execution.RootEventID)
	queueItem.RunID = run.ID
	require.NoError(t, database.Conn().Save(queueItem).Error)

	openWork, err := run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.True(t, openWork.HasQueueItems)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasPendingEvents)

	require.NoError(t, queueItem.Delete(database.Conn()))
	openWork, err = run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.False(t, openWork.HasQueueItems)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasPendingEvents)
}

func Test__CanvasRun__FindOpenWork__ActiveExecution(t *testing.T) {
	run, execution := setupRunWithExecution(t)

	for _, state := range []string{
		models.CanvasNodeExecutionStatePending,
		models.CanvasNodeExecutionStateStarted,
	} {
		require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
			"state":      state,
			"updated_at": time.Now(),
		}).Error)

		openWork, err := run.FindOpenWork(database.DB(t.Context()))
		require.NoError(t, err)
		assert.True(t, openWork.HasActiveExecutions, "expected active execution for state %s", state)
		assert.False(t, openWork.HasQueueItems)
		assert.False(t, openWork.HasPendingEvents)
	}
}

func Test__CanvasRun__CalculateResult__FailedTakesPrecedenceOverCancelled(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultCancelled,
		"updated_at": time.Now(),
	}).Error)

	failedExecution := createExecutionForRun(t, run, execution.RootEventID, "node-2")
	require.NoError(t, database.Conn().Model(failedExecution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultFailed,
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultFailed, result)
}

func Test__CanvasRun__CalculateResult__Cancelled(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultCancelled,
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultCancelled, result)
}

func Test__CanvasRun__CalculateResult__Passed(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, result)
}

func Test__ListCanvasRuns__StatesAndResultsAreOredWithinWorkflow(t *testing.T) {
	r := support.Setup(t)

	canvasA, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)
	canvasB, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	runningA := createRunWithState(t, canvasA.ID, models.CanvasRunStateStarted, "")
	passedA := createRunWithState(t, canvasA.ID, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
	createRunWithState(t, canvasA.ID, models.CanvasRunStateFinished, models.CanvasRunResultFailed)

	// Runs in workflow B should never leak in even when their result matches the filter.
	createRunWithState(t, canvasB.ID, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
	createRunWithState(t, canvasB.ID, models.CanvasRunStateStarted, "")

	runs, err := models.ListCanvasRuns(canvasA.ID, 50, nil, models.CanvasRunFilters{
		States:  []string{models.CanvasRunStateStarted},
		Results: []string{models.CanvasRunResultPassed},
	})
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool, len(runs))
	for _, run := range runs {
		assert.Equal(t, canvasA.ID, run.WorkflowID, "filter must not leak runs from other workflows")
		ids[run.ID] = true
	}
	assert.True(t, ids[runningA.ID], "expected the running run to be included")
	assert.True(t, ids[passedA.ID], "expected the passed run to be included")
	assert.Len(t, runs, 2, "expected only running+passed runs from canvas A")
}

func createRunWithState(t *testing.T, workflowID uuid.UUID, state, result string) *models.CanvasRun {
	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		NodeID:     "trigger",
		VersionID:  liveVersion.ID,
		State:      state,
		Result:     result,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	if state == models.CanvasRunStateFinished {
		run.FinishedAt = &now
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func setupRunWithExecution(t *testing.T) (*models.CanvasRun, *models.CanvasNodeExecution) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "node-1", Type: models.NodeTypeComponent},
			{NodeID: "node-2", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)
	execution := createExecutionForRun(t, run, rootEvent.ID, "node-1")
	return run, execution
}

func createRunForRootEvent(t *testing.T, rootEvent *models.CanvasEvent) *models.CanvasRun {
	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}

		return rootEvent.RoutedInTransaction(tx)
	}))

	require.Equal(t, "trigger", run.NodeID)
	return run
}

func createExecutionForRun(t *testing.T, run *models.CanvasRun, rootEventID uuid.UUID, nodeID string) *models.CanvasNodeExecution {
	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    run.WorkflowID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		RunID:         run.ID,
		EventID:       rootEventID,
		State:         models.CanvasNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func subRunTestCanvasNodes(entrypoints ...string) []models.CanvasNode {
	nodes := []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger},
	}
	for _, nodeID := range entrypoints {
		nodes = append(nodes, models.CanvasNode{
			NodeID: nodeID,
			Type:   models.NodeTypeComponent,
		})
	}
	return nodes
}

func Test__ValidateSubRunCreationInTransaction__SameWorkflowDoesNotIncreaseCrossWorkflowDepth(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("run1", "run2"),
		nil,
	)

	parentRun := createSubRun(t, canvas.ID, "run1", nil, nil, nil)
	err := models.ValidateSubRunCreation(
		database.Conn(),
		parentRun.ID,
		canvas.ID,
		"run2",
		8,
	)
	require.NoError(t, err)
}

func Test__ValidateSubRunCreationInTransaction__CrossWorkflowDepthAcrossApps(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvasA, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("runA"),
		nil,
	)
	canvasB, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("runB"),
		nil,
	)
	canvasC, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("runC"),
		nil,
	)

	runA := createSubRun(t, canvasA.ID, "runA", nil, nil, nil)
	runB := createSubRun(t, canvasB.ID, "runB", &runA.ID, &canvasA.ID, nil)
	runC := createSubRun(t, canvasC.ID, "runC", &runB.ID, &canvasB.ID, nil)

	err := models.ValidateSubRunCreation(
		database.Conn(),
		runC.ID,
		uuid.New(),
		"runD",
		8,
	)
	require.NoError(t, err)

	err = models.ValidateSubRunCreation(
		database.Conn(),
		runC.ID,
		uuid.New(),
		"runD",
		2,
	)
	require.ErrorIs(t, err, models.ErrSubRunCrossWorkflowDepthExceeded)
}

func Test__ValidateSubRunCreationInTransaction__WorkflowCycleAcrossApps(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvasA, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("runA"),
		nil,
	)
	canvasB, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("runB"),
		nil,
	)
	canvasC, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("runC"),
		nil,
	)

	runA := createSubRun(t, canvasA.ID, "", nil, nil, nil)
	runB := createSubRun(t, canvasB.ID, "runB", &runA.ID, &canvasA.ID, nil)
	runC := createSubRun(t, canvasC.ID, "runC", &runB.ID, &canvasB.ID, nil)

	err := models.ValidateSubRunCreation(
		database.Conn(),
		runC.ID,
		canvasA.ID,
		"runA",
		8,
	)
	require.ErrorIs(t, err, models.ErrSubRunWorkflowCycle)
}

func Test__ValidateSubRunCreationInTransaction__EntrypointCycleWithinWorkflow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("run1", "run2"),
		nil,
	)

	rootRun := createSubRun(t, canvas.ID, "", nil, nil, nil)
	run2InChain := createSubRun(t, canvas.ID, "run2", &rootRun.ID, &canvas.ID, nil)
	parentRun := createSubRun(t, canvas.ID, "run1", &run2InChain.ID, &canvas.ID, nil)

	err := models.ValidateSubRunCreation(
		database.Conn(),
		parentRun.ID,
		canvas.ID,
		"run2",
		8,
	)
	require.ErrorIs(t, err, models.ErrSubRunEntrypointCycle)
}

func Test__ValidateSubRunCreationInTransaction__SiblingSubRunsAllowRepeatedEntrypoint(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		subRunTestCanvasNodes("forEach", "item"),
		nil,
	)

	parentRun := createSubRun(t, canvas.ID, "forEach", nil, nil, nil)
	createSubRun(t, canvas.ID, "item", &parentRun.ID, &canvas.ID, nil)

	err := models.ValidateSubRunCreation(
		database.Conn(),
		parentRun.ID,
		canvas.ID,
		"item",
		8,
	)
	require.NoError(t, err)
}

func createSubRun(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	parentRunID *uuid.UUID,
	parentWorkflowID *uuid.UUID,
	parentExecutionID *uuid.UUID,
) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	effectiveNodeID := nodeID
	if effectiveNodeID == "" {
		effectiveNodeID = "trigger"
	}

	run := models.CanvasRun{
		ID:                uuid.New(),
		WorkflowID:        workflowID,
		NodeID:            effectiveNodeID,
		VersionID:         liveVersion.ID,
		ParentRunID:       parentRunID,
		ParentWorkflowID:  parentWorkflowID,
		ParentExecutionID: parentExecutionID,
		State:             models.CanvasRunStatePending,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func TestShallowMergeObjects(t *testing.T) {
	t.Run("merges disjoint keys", func(t *testing.T) {
		merged := models.ShallowMergeObjects(
			map[string]any{"a": 1},
			map[string]any{"b": 2},
		)

		assert.Equal(t, map[string]any{"a": 1, "b": 2}, merged)
	})

	t.Run("patch replaces top-level keys", func(t *testing.T) {
		merged := models.ShallowMergeObjects(
			map[string]any{"deploy": map[string]any{"status": "running"}},
			map[string]any{"deploy": map[string]any{"status": "failed", "url": "https://example.com"}},
		)

		assert.Equal(t, map[string]any{
			"deploy": map[string]any{"status": "failed", "url": "https://example.com"},
		}, merged)
	})

	t.Run("empty patch returns copy of base", func(t *testing.T) {
		base := map[string]any{"a": 1}
		merged := models.ShallowMergeObjects(base, map[string]any{})

		assert.Equal(t, base, merged)
	})
}

func Test__CanvasRun__AssignRunOutput(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	run, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)

	t.Run("merges output on the run", func(t *testing.T) {
		err := run.AssignRunOutput(database.Conn(), map[string]any{
			"deploy": map[string]any{"id": "d-1"},
		}, 1024)
		require.NoError(t, err)

		err = run.AssignRunOutput(database.Conn(), map[string]any{
			"test": map[string]any{"passed": true},
		}, 1024)
		require.NoError(t, err)

		updated, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
		require.NoError(t, err)

		output := updated.Output.Data().(map[string]any)
		assert.Equal(t, map[string]any{
			"deploy": map[string]any{"id": "d-1"},
			"test":   map[string]any{"passed": true},
		}, output)
	})

	t.Run("rejects output larger than max size", func(t *testing.T) {
		otherRun, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
		require.NoError(t, err)

		err = otherRun.AssignRunOutput(database.Conn(), map[string]any{
			"payload": strings.Repeat("a", 128),
		}, 32)
		require.ErrorIs(t, err, models.ErrRunOutputTooLarge)
	})

	t.Run("rejects nil patch", func(t *testing.T) {
		err := run.AssignRunOutput(database.Conn(), nil, 1024)
		require.ErrorIs(t, err, models.ErrRunOutputPatchInvalid)
	})
}

func Test__CanvasRun__AddError(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	run, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)

	t.Run("appends error entries on the run", func(t *testing.T) {
		err := run.AddError(database.Conn(), "pipeline failed", 1024)
		require.NoError(t, err)

		err = run.AddError(database.Conn(), "tests failed", 1024)
		require.NoError(t, err)

		updated, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
		require.NoError(t, err)
		assert.Equal(t, []models.RunError{
			{Message: "pipeline failed"},
			{Message: "tests failed"},
		}, []models.RunError(updated.Errors))
	})

	t.Run("rejects empty message", func(t *testing.T) {
		otherRun, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
		require.NoError(t, err)

		err = otherRun.AddError(database.Conn(), "", 1024)
		require.ErrorIs(t, err, models.ErrRunErrorMessageRequired)
	})

	t.Run("rejects errors larger than max size", func(t *testing.T) {
		otherRun, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
		require.NoError(t, err)

		err = otherRun.AddError(database.Conn(), strings.Repeat("a", 128), 32)
		require.ErrorIs(t, err, models.ErrRunErrorsTooLarge)
	})
}

func Test__CanvasRun__CalculateResult__WithErrors(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	run, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)

	err = run.AddError(database.Conn(), "pipeline failed", 1024)
	require.NoError(t, err)

	updated, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)

	result, err := updated.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultFailed, result)
}

func Test__CanvasRun__ErrorMessages(t *testing.T) {
	run := models.CanvasRun{}
	assert.Nil(t, run.ErrorMessages())

	run.Errors = []models.RunError{
		{Message: "pipeline failed"},
		{Message: "tests failed"},
	}
	assert.Equal(t, []string{"pipeline failed", "tests failed"}, run.ErrorMessages())
}
