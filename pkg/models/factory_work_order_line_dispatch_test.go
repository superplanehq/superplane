package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryLine__Dispatch__SnapshotsLineAndStartsStepZero(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}
	require.NoError(t, line.Update(db, nil, steps))

	var dispatch *models.FactoryWorkOrderLineDispatch
	var result *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, result, dispatchErr = line.Dispatch(tx, order)
		return dispatchErr
	}))

	assert.Equal(t, order.ID, dispatch.WorkOrderID)
	assert.Equal(t, line.ID, dispatch.LineID)
	assert.Equal(t, line.Name, dispatch.LineName)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, dispatch.State)
	assert.Empty(t, dispatch.Result)
	assert.Nil(t, dispatch.FinishedAt)
	require.Len(t, []models.FactoryLineStep(dispatch.Steps), 2)
	assert.Equal(t, firstApp.ID, dispatch.Steps[0].AppID)
	assert.Equal(t, secondApp.ID, dispatch.Steps[1].AppID)

	require.NotNil(t, result)
	execution := result.Execution
	assert.Equal(t, dispatch.ID, execution.LineDispatchID)
	assert.Equal(t, 0, execution.StepIndex)
	assert.Equal(t, firstApp.Name, execution.StepName)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, execution.Status)
}

func Test__FactoryLine__DispatchFrom__StartsRequestedStep(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}))

	var result *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		_, result, dispatchErr = line.DispatchFrom(tx, order, 1)
		return dispatchErr
	}))

	require.NotNil(t, result)
	assert.Equal(t, 1, result.Execution.StepIndex)
	assert.Equal(t, secondApp.Name, result.Execution.StepName)
}

func Test__FactoryWorkOrder__AbandonActiveLineDispatch(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
	})
	require.NoError(t, err)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, dispatchErr := line.Dispatch(tx, order)
		return dispatchErr
	}))

	require.NoError(t, order.AbandonActiveLineDispatch(db))

	_, err = order.FindActiveLineDispatch(db)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__FactoryLine__Dispatch__RejectsLineWithNoSteps(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "empty", nil)
	require.NoError(t, err)

	err = db.Transaction(func(tx *gorm.DB) error {
		_, _, dispatchErr := line.Dispatch(tx, order)
		return dispatchErr
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrFactoryLineHasNoSteps)
}

// Test__FactoryWorkOrderLineDispatch__StartStep__UsesSnapshotNotLiveLine
// covers acceptance criterion 3 at the model layer (see also the
// run_finalizer advancement tests for the end-to-end path): once a
// dispatch snapshots the line's steps, editing the live line doesn't
// change what StartStep resolves for a given step index.
func Test__FactoryWorkOrderLineDispatch__StartStep__UsesSnapshotNotLiveLine(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}))

	var dispatch *models.FactoryWorkOrderLineDispatch
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, _, dispatchErr = line.Dispatch(tx, order)
		return dispatchErr
	}))

	// Replace the live line's steps entirely with a different app at index 1.
	thirdApp, thirdEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two-replaced", "start-replaced")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: thirdApp.ID, Entrypoint: thirdEntry},
	}))

	var result *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var startErr error
		result, startErr = dispatch.StartStep(tx, order, 1)
		return startErr
	}))

	assert.Equal(t, secondApp.ID, result.Run.WorkflowID,
		"the dispatch's own snapshot still points at the originally dispatched app")
	assert.Equal(t, secondApp.Name, result.Execution.StepName)
}

func Test__FactoryWorkOrderLineDispatch__Finish(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "line", nil)
	require.NoError(t, err)

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

	require.NoError(t, dispatch.Finish(db, models.CanvasRunResultFailed))
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, dispatch.State)
	assert.Equal(t, models.CanvasRunResultFailed, dispatch.Result)
	require.NotNil(t, dispatch.FinishedAt)
	assert.WithinDuration(t, time.Now(), *dispatch.FinishedAt, 5*time.Second)

	reloaded, err := models.FindWorkOrderLineDispatch(db, dispatch.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State)
	assert.Equal(t, models.CanvasRunResultFailed, reloaded.Result)

	// Finishing an already-finished dispatch is a no-op, not an error, and
	// doesn't clobber the recorded result.
	require.NoError(t, dispatch.Finish(db, models.CanvasRunResultPassed))
	assert.Equal(t, models.CanvasRunResultFailed, dispatch.Result)
}

func Test__FactoryWorkOrder__FindActiveLineDispatch(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = order.FindActiveLineDispatch(db)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "no dispatch exists yet")

	line, err := factory.CreateLine(db, "line", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, dispatch.ID, active.ID)

	require.NoError(t, dispatch.Finish(db, models.CanvasRunResultPassed))

	_, err = order.FindActiveLineDispatch(db)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "finished dispatches don't count as active")
}

func Test__FactoryWorkOrder__RetryLineStep__ReusesDispatchWithEarlierSteps(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Improve AGENTS.md", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "plan", "start-plan")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start-implement")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	})
	require.NoError(t, err)

	var full *models.FactoryWorkOrderLineDispatch
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		full, _, dispatchErr = line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		if _, startErr := full.EnqueueOrStartStep(tx, order, 1); startErr != nil {
			return startErr
		}
		return full.Finish(tx, models.CanvasRunResultFailed)
	}))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, dispatchErr := line.DispatchFrom(tx, order, 1)
		if dispatchErr != nil {
			return dispatchErr
		}
		return order.AbandonActiveLineDispatch(tx)
	}))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, retryErr := order.RetryLineStep(tx, line, 1)
		return retryErr
	}))

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, full.ID, active.ID)

	byOrder, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, []uuid.UUID{order.ID})
	require.NoError(t, err)
	var planCount int
	for _, record := range byOrder[order.ID] {
		if record.ID != full.ID {
			continue
		}
		for _, execution := range record.Executions {
			if execution.StepIndex == 0 {
				planCount++
			}
		}
	}
	assert.Equal(t, 1, planCount)
}

func Test__FactoryWorkOrder__RetryLineStep__SettlesInFlightStepBeforeRerun(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Improve AGENTS.md", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "plan", "start-plan")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start-implement")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	})
	require.NoError(t, err)

	var dispatch *models.FactoryWorkOrderLineDispatch
	var inFlight *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, _, dispatchErr = line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		result, startErr := dispatch.EnqueueOrStartStep(tx, order, 1)
		if startErr != nil {
			return startErr
		}
		inFlight = result.Execution
		return nil
	}))

	require.NotEqual(t, models.FactoryWorkOrderExecutionStatusFinished, inFlight.Status)
	require.NotNil(t, inFlight.RunID)

	var retry *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var retryErr error
		retry, _, retryErr = order.RetryLineStep(tx, line, 1)
		return retryErr
	}))

	require.NotNil(t, retry)
	require.NotNil(t, retry.Execution)
	assert.NotEqual(t, inFlight.ID, retry.Execution.ID)
	assert.Equal(t, 1, retry.Execution.StepIndex)
	assert.NotEqual(t, models.FactoryWorkOrderExecutionStatusFinished, retry.Execution.Status)

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, dispatch.ID, active.ID)

	settled, err := models.FindWorkOrderExecutionByRunID(db, *inFlight.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, settled.Status)
	assert.Equal(t, models.CanvasRunResultCancelled, settled.Result)
}

func Test__FactoryWorkOrder__RetryLineStep__AdmitsQueuedWorkForFreedSlot(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "plan", "start-plan")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start-implement")
	limit := 1
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry, MaxParallelism: &limit},
	})
	require.NoError(t, err)

	running, err := factory.CreateWorkOrder(db, "Running", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = running.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	queued, err := factory.CreateWorkOrder(db, "Queued", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = queued.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)

	var queuedDispatch *models.FactoryWorkOrderLineDispatch
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		runningDispatch, _, dispatchErr := line.Dispatch(tx, running)
		if dispatchErr != nil {
			return dispatchErr
		}
		if _, startErr := runningDispatch.EnqueueOrStartStep(tx, running, 1); startErr != nil {
			return startErr
		}

		var queuedErr error
		queuedDispatch, _, queuedErr = line.Dispatch(tx, queued)
		if queuedErr != nil {
			return queuedErr
		}
		result, startErr := queuedDispatch.EnqueueOrStartStep(tx, queued, 1)
		if startErr != nil {
			return startErr
		}
		if result.QueueItem == nil {
			return fmt.Errorf("expected the second order to wait for the step slot")
		}
		return nil
	}))

	var started []*models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var retryErr error
		_, started, retryErr = running.RetryLineStep(tx, line, 1)
		return retryErr
	}))

	byDispatch, err := models.ListFactoryWorkOrderQueueItemsByLineDispatchIDs(db, []uuid.UUID{queuedDispatch.ID})
	require.NoError(t, err)
	_, stillQueued := byDispatch[queuedDispatch.ID]
	assert.False(t, stillQueued, "the freed slot must admit the waiting dispatch")

	executions, err := models.ListFactoryWorkOrderExecutionsByLineDispatchIDs(db, []uuid.UUID{queuedDispatch.ID})
	require.NoError(t, err)
	var implement *models.FactoryWorkOrderExecution
	for i := range executions[queuedDispatch.ID] {
		execution := executions[queuedDispatch.ID][i]
		if execution.StepIndex == 1 {
			implement = &execution.FactoryWorkOrderExecution
		}
	}
	require.NotNil(t, implement)
	assert.NotEqual(t, models.FactoryWorkOrderExecutionStatusFinished, implement.Status)

	var admittedRun *models.CanvasRun
	for _, result := range started {
		if result.Execution != nil && result.Execution.WorkOrderID == queued.ID {
			admittedRun = result.Run
		}
	}
	require.NotNil(t, admittedRun, "admitted waiters must be in the started list so dispatch can publish them")
}
