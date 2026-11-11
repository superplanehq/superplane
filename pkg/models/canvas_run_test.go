package models_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__ListExpiredFinishedRuns__BoundariesAndOrdering(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	now := time.Now().UTC().Truncate(time.Microsecond)
	var expected []uuid.UUID
	for _, days := range []int{30, 7, 0, -1, -2} {
		org, err := models.CreateOrganizationInTransaction(db, support.RandomName("org"), support.RandomName("org"))
		require.NoError(t, err)
		require.NoError(t, db.Model(org).Update("usage_retention_window_days", days).Error)
		canvas, _ := support.CreateCanvas(t, org.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, nil)
		cutoff := now.AddDate(0, 0, -days)
		for _, offset := range []time.Duration{time.Microsecond, 0, -time.Microsecond} {
			run := createRunWithState(t, canvas.ID, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
			require.NoError(t, db.Model(run).Update("finished_at", cutoff.Add(offset)).Error)
			if days > 0 && offset < 0 {
				expected = append(expected, run.ID)
			}
		}
		unfinished := createRunWithState(t, canvas.ID, models.CanvasRunStateStarted, "")
		require.NoError(t, db.Model(unfinished).Update("finished_at", now.AddDate(0, 0, -100)).Error)
		missingTime := createRunWithState(t, canvas.ID, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
		require.NoError(t, db.Model(missingTime).Update("finished_at", nil).Error)
		if days == -2 {
			require.NoError(t, db.Model(org).Update("usage_retention_window_days", nil).Error)
		}
	}
	// A non-UTC caller must use the same timestamp boundary.
	referenceTime := now.In(time.FixedZone("caller", 5*60*60))
	for _, limit := range []int{1, 0, -1} {
		runs, err := models.ListExpiredFinishedRuns(db, referenceTime, limit)
		require.NoError(t, err)
		var ids []uuid.UUID
		for _, run := range runs {
			ids = append(ids, run.ID)
		}
		want := expected
		if limit > 0 {
			want = expected[:limit]
		}
		require.Equal(t, want, ids)
	}
}

func Test__ExpiredFinishedRun__RevalidatesActiveWorkAndRetention(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	now := time.Now().UTC().Truncate(time.Microsecond)
	var canvas models.Canvas
	require.NoError(t, db.First(&canvas, "id = ?", run.WorkflowID).Error)
	require.NoError(t, db.Model(&models.Organization{}).Where("id = ?", canvas.OrganizationID).Update("usage_retention_window_days", 7).Error)
	require.NoError(t, db.Model(run).Updates(map[string]any{"state": models.CanvasRunStateFinished, "finished_at": now.AddDate(0, 0, -8)}).Error)
	require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateFinished).Error)
	runs, err := models.ListExpiredFinishedRuns(db, now, 1)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, run.ID, runs[0].ID)
	for _, state := range []string{models.CanvasNodeExecutionStatePending, models.CanvasNodeExecutionStateStarted, models.CanvasNodeExecutionStateCancelling} {
		require.NoError(t, db.Model(execution).Update("state", state).Error)
		runs, err = models.ListExpiredFinishedRuns(db, now, 1)
		require.NoError(t, err)
		require.Empty(t, runs)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			locked, err := models.LockExpiredFinishedRun(tx, now, run.ID)
			require.NoError(t, err)
			require.Nil(t, locked)
			return nil
		}))
	}
	require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateFinished).Error)
	for _, days := range []int{30, 0} {
		require.NoError(t, db.Model(&models.Organization{}).Where("id = ?", canvas.OrganizationID).Update("usage_retention_window_days", days).Error)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			locked, err := models.LockExpiredFinishedRun(tx, now, run.ID)
			require.NoError(t, err)
			require.Nil(t, locked)
			return nil
		}))
	}
	require.NoError(t, db.Model(&models.Organization{}).Where("id = ?", canvas.OrganizationID).Update("usage_retention_window_days", 7).Error)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	locked, err := models.LockExpiredFinishedRun(tx, now, run.ID)
	require.NoError(t, err)
	require.NotNil(t, locked)
	require.NoError(t, db.Transaction(func(other *gorm.DB) error {
		locked, err := models.LockExpiredFinishedRun(other, now, run.ID)
		require.NoError(t, err)
		require.Nil(t, locked)
		return nil
	}))
}

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

func Test__CanvasRun__CalculateResult__CompletionRequested(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultCancelled,
		"updated_at": time.Now(),
	}).Error)

	run.State = models.CanvasRunStateCancelling
	run.Result = models.CanvasRunResultPassed
	result, err := run.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, result)
}

func Test__CanvasRun__RequestCompletion__StopsRunAndIsIdempotent(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())

	for range 2 {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			_, err := run.RequestCompletion(tx, nil)
			return err
		}))
	}

	updatedRun, err := models.FindCanvasRunInTransaction(db, run.WorkflowID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)

	updatedExecution, err := models.FindNodeExecutionInTransaction(db, run.WorkflowID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
}

func Test__CanvasRun__RequestCompletion__DoesNotRewriteFinishedRun(t *testing.T) {
	for _, storedResult := range []string{
		models.CanvasRunResultCancelled,
		models.CanvasRunResultPassed,
	} {
		t.Run(storedResult, func(t *testing.T) {
			r := support.Setup(t)
			canvas, _ := support.CreateCanvas(
				t,
				r.Organization.ID,
				r.User,
				[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
				[]models.Edge{},
			)
			run := createRunWithState(t, canvas.ID, models.CanvasRunStateFinished, storedResult)
			db := database.DB(t.Context())

			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				_, err := run.RequestCompletion(tx, nil)
				return err
			}))

			updatedRun, err := models.FindCanvasRunInTransaction(db, run.WorkflowID, run.ID)
			require.NoError(t, err)
			assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
			assert.Equal(t, storedResult, updatedRun.Result)
		})
	}
}

func Test__CanvasRun__RequestCancellation__OverridesCompletionRequest(t *testing.T) {
	run, _ := setupRunWithExecution(t)
	db := database.DB(t.Context())

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if _, err := run.RequestCompletion(tx, nil); err != nil {
			return err
		}
		_, err := run.RequestCancellation(tx, nil)
		return err
	}))

	updatedRun, err := models.FindCanvasRunInTransaction(db, run.WorkflowID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)
	assert.Empty(t, updatedRun.Result)

	result, err := updatedRun.CalculateResult(db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultCancelled, result)
}

func Test__CanvasRun__CancellationDrainMakesBoundedProgress(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	for i := 0; i < 101; i++ {
		createExecutionForRun(t, run, execution.RootEventID, "node-1")
		queue := support.CreateQueueItem(t, run.WorkflowID, "node-1", execution.RootEventID, execution.RootEventID)
		require.NoError(t, db.Model(queue).Update("run_id", run.ID).Error)
	}
	var result *models.RunCancellationResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = run.RequestCancellationBatch(tx)
		return err
	}))
	require.Len(t, result.Drain.DeletedQueueItems, 100)
	open, err := run.FindOpenWork(db)
	require.NoError(t, err)
	require.True(t, open.HasQueueItems)
	eligible, err := run.ListExecutionsInStates(db, []string{models.CanvasNodeExecutionStatePending, models.CanvasNodeExecutionStateStarted})
	require.NoError(t, err)
	require.Len(t, eligible, 2)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = run.RequestCancellationBatch(tx)
		return err
	}))
	require.Len(t, result.Drain.DeletedQueueItems, 1)
	open, err = run.FindOpenWork(db)
	require.NoError(t, err)
	require.False(t, open.HasQueueItems)
	eligible, err = run.ListExecutionsInStates(db, []string{models.CanvasNodeExecutionStatePending, models.CanvasNodeExecutionStateStarted})
	require.NoError(t, err)
	require.Empty(t, eligible)
}

func Test__CanvasRun__OrdinaryTerminationDrainsAllWork(t *testing.T) {
	for _, test := range []struct {
		name       string
		completion bool
	}{{name: "cancellation"}, {name: "completion", completion: true}} {
		t.Run(test.name, func(t *testing.T) {
			run, execution := setupRunWithExecution(t)
			db := database.DB(t.Context())
			for i := 0; i < 101; i++ {
				createExecutionForRun(t, run, execution.RootEventID, "node-1")
				queue := support.CreateQueueItem(t, run.WorkflowID, "node-1", execution.RootEventID, execution.RootEventID)
				require.NoError(t, db.Model(queue).Update("run_id", run.ID).Error)
				event := support.EmitCanvasEventForNode(t, run.WorkflowID, "node-1", "default", nil)
				require.NoError(t, db.Model(event).Update("run_id", run.ID).Error)
			}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				if test.completion {
					_, err := run.RequestCompletion(tx, nil)
					return err
				}
				_, err := run.RequestCancellation(tx, nil)
				return err
			}))
			eligible, err := run.ListExecutionsInStates(db, []string{models.CanvasNodeExecutionStatePending, models.CanvasNodeExecutionStateStarted})
			require.NoError(t, err)
			require.Empty(t, eligible)
			open, err := run.FindOpenWork(db)
			require.NoError(t, err)
			require.False(t, open.HasQueueItems)
			require.False(t, open.HasPendingEvents)
		})
	}
}

func Test__CanvasRun__StaleCancellationPreservesFinishedExecution(t *testing.T) {
	_, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Model(execution).Updates(map[string]any{"state": models.CanvasNodeExecutionStateFinished, "result": models.CanvasNodeExecutionResultPassed}).Error)
	require.NoError(t, execution.RequestCancellation(db, nil))
	stored, err := models.FindNodeExecutionInTransaction(db, execution.WorkflowID, execution.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionStateFinished, stored.State)
	require.Equal(t, models.CanvasNodeExecutionResultPassed, stored.Result)
}

func Test__CanvasRun__RequestCancellation__StopsDescendantsOfFinishedExecutions(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"state":  models.CanvasNodeExecutionStateFinished,
		"result": models.CanvasNodeExecutionResultPassed,
	}).Error)
	child := createSubRun(t, run.WorkflowID, "node-1", &run.ID, &run.WorkflowID, &execution.ID)
	require.NoError(t, db.Model(child).Updates(map[string]any{
		"state":  models.CanvasRunStateFinished,
		"result": models.CanvasRunResultPassed,
	}).Error)
	grandchild := createSubRun(t, run.WorkflowID, "node-1", &child.ID, &run.WorkflowID, nil)
	unrelated := createSubRun(t, run.WorkflowID, "node-1", nil, nil, nil)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := run.RequestCancellation(tx, nil)
		return err
	}))

	for _, expected := range []struct {
		run    *models.CanvasRun
		state  string
		result string
	}{
		{run, models.CanvasRunStateCancelling, ""},
		{child, models.CanvasRunStateFinished, models.CanvasRunResultPassed},
		{grandchild, models.CanvasRunStateCancelling, ""},
		{unrelated, models.CanvasRunStatePending, ""},
	} {
		loaded, err := models.FindUnscopedCanvasRun(db, expected.run.ID)
		require.NoError(t, err)
		assert.Equal(t, expected.state, loaded.State)
		assert.Equal(t, expected.result, loaded.Result)
	}
}

func Test__CanvasRun__RequestCompletion__DoesNotOverrideCancellationRequest(t *testing.T) {
	run, _ := setupRunWithExecution(t)
	db := database.DB(t.Context())

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if _, err := run.RequestCancellation(tx, nil); err != nil {
			return err
		}
		_, err := run.RequestCompletion(tx, nil)
		return err
	}))

	updatedRun, err := models.FindCanvasRunInTransaction(db, run.WorkflowID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)
	assert.Empty(t, updatedRun.Result)

	result, err := updatedRun.CalculateResult(db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultCancelled, result)
}

func Test__CanvasRun__CalculateResult__UnconsumedFailedChannelFailsRun(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	cancelled := createExecutionForRun(t, run, execution.RootEventID, "node-2")
	require.NoError(t, db.Model(cancelled).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultCancelled,
		"updated_at": time.Now(),
	}).Error)

	support.EmitCanvasEventForNode(t, run.WorkflowID, execution.NodeID, runner.FailedOutputChannel, &execution.ID)

	result, err := run.CalculateResult(db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultFailed, result)
}

func Test__CanvasRun__CalculateResult__ConsumedFailedChannelPassesWhenHandlerPasses(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	failedEvent := support.EmitCanvasEventForNode(t, run.WorkflowID, execution.NodeID, runner.FailedOutputChannel, &execution.ID)
	handler := createExecutionForRun(t, run, execution.RootEventID, "node-2")
	require.NoError(t, db.Model(handler).Updates(map[string]any{
		"event_id":   failedEvent.ID,
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, result)
}

func Test__CanvasRun__CalculateResult__FailedChannelHandledBySharedExecutionPasses(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	passedEvent := support.EmitCanvasEventForNode(t, run.WorkflowID, execution.NodeID, "default", &execution.ID)
	firstFailed := support.EmitCanvasEventForNode(t, run.WorkflowID, execution.NodeID, runner.FailedOutputChannel, &execution.ID)
	secondFailed := support.EmitCanvasEventForNode(t, run.WorkflowID, execution.NodeID, runner.FailedOutputChannel, &execution.ID)
	shared := createExecutionForRun(t, run, execution.RootEventID, "node-2")
	require.NoError(t, db.Model(shared).Updates(map[string]any{
		"event_id": passedEvent.ID,
		"state":    models.CanvasNodeExecutionStateFinished,
		"result":   models.CanvasNodeExecutionResultPassed,
		"metadata": datatypes.NewJSONType(map[string]any{
			"eventIDs": []string{
				passedEvent.ID.String(),
				firstFailed.ID.String(),
				secondFailed.ID.String(),
			},
		}),
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, result)
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

	runs, err := models.ListCanvasRuns(database.DB(t.Context()), canvasA.ID, 50, nil, models.CanvasRunFilters{
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

	t.Run("concurrent appends keep both messages", func(t *testing.T) {
		otherRun, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
		require.NoError(t, err)

		messages := []string{"pipeline failed", "tests failed"}
		var started sync.WaitGroup
		started.Add(len(messages))

		var wg sync.WaitGroup
		errs := make([]error, len(messages))
		for i, message := range messages {
			wg.Add(1)
			go func(idx int, message string) {
				defer wg.Done()
				errs[idx] = database.Conn().Transaction(func(tx *gorm.DB) error {
					var current models.CanvasRun
					if err := tx.Where("id = ?", otherRun.ID).First(&current).Error; err != nil {
						started.Done()
						return err
					}

					started.Done()
					started.Wait()
					return current.AddError(tx, message, 1024)
				})
			}(i, message)
		}
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}

		updated, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, otherRun.ID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []models.RunError{
			{Message: "pipeline failed"},
			{Message: "tests failed"},
		}, []models.RunError(updated.Errors))
	})

	t.Run("does not block child event inserts while the lock is held", func(t *testing.T) {
		otherRun, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "trigger", models.CanvasRunStateStarted, "")
		require.NoError(t, err)

		holding := make(chan struct{})
		release := make(chan struct{})
		holdErr := make(chan error, 1)

		go func() {
			holdErr <- database.Conn().Transaction(func(tx *gorm.DB) error {
				if err := otherRun.AddError(tx, "pipeline failed", 1024); err != nil {
					return err
				}

				close(holding)
				<-release
				return nil
			})
		}()

		select {
		case <-holding:
		case err := <-holdErr:
			require.NoError(t, err)
			t.Fatal("lock holder transaction ended before the child insert")
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting to hold the run lock")
		}

		insertErr := database.Conn().Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SET LOCAL lock_timeout = '500ms'").Error; err != nil {
				return err
			}

			now := time.Now()
			event := models.CanvasEvent{
				WorkflowID: canvas.ID,
				NodeID:     "trigger",
				Channel:    "default",
				Data:       models.NewJSONValue(map[string]any{"key": "value"}),
				State:      models.CanvasEventStatePending,
				RunID:      otherRun.ID,
				CreatedAt:  &now,
			}
			return tx.Create(&event).Error
		})
		require.NoError(t, insertErr)

		close(release)
		require.NoError(t, <-holdErr)
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
