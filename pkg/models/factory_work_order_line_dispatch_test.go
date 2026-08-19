package models_test

import (
	"testing"
	"time"

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
