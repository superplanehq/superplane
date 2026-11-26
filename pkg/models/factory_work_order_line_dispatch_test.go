package models_test

import (
	"fmt"
	"os"
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
	require.NoError(t, line.Update(db, nil, steps, nil))

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

func Test__FactoryLine__Dispatch__StartsCustomStageBeforeLineStep(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	phaseApp, phaseEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: phaseApp.ID, Entrypoint: phaseEntry,
	}})
	require.NoError(t, err)

	stageID := uuid.New()
	customStage := models.FactoryLineStage{
		ID: stageID, Key: "custom-" + stageID.String(), Name: "QA", Kind: models.FactoryLineStagePhase,
	}
	stages := append([]models.FactoryLineStage{line.StagesValue()[0], customStage}, line.StagesValue()[1:]...)
	stages[2].AgentDisplayKey = customStage.Key
	eventCanvas := support.CreateFactoryCanvas(t, r, factory.ID, "PR checks")
	require.NoError(t, eventCanvas.SetColumnKey(db, "verify"))
	require.NoError(t, eventCanvas.SetDisplayColumn(db, customStage.Key))
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	first, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Lint", "lint")
	second, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Test", "test")
	require.NoError(t, first.SetColumnKey(db, customStage.Key))
	require.NoError(t, second.SetColumnKey(db, customStage.Key))
	manual, manualEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Manual QA", "manual")
	require.NoError(t, manual.SetColumnKey(db, customStage.Key))
	manualVersion, err := models.FindLiveCanvasVersionInTransaction(db, manual.ID)
	require.NoError(t, err)
	manualVersion.Nodes[0].Metadata = map[string]any{models.FactoryAppTemplateMetadataKey: map[string]any{"startMode": "manual"}}
	require.NoError(t, db.Save(manualVersion).Error)

	var dispatch *models.FactoryWorkOrderLineDispatch
	var results []*models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, results, dispatchErr = line.DispatchStage(tx, order)
		return dispatchErr
	}))

	require.Len(t, dispatch.StagesValue(), len(stages))
	require.Len(t, []models.FactoryLineStep(dispatch.Steps), 3)
	assert.Equal(t, []uuid.UUID{first.ID, second.ID, phaseApp.ID}, []uuid.UUID{
		dispatch.Steps[0].AppID,
		dispatch.Steps[1].AppID,
		dispatch.Steps[2].AppID,
	})
	assert.Equal(t, customStage.Key, dispatch.Steps[0].StageKey)
	assert.Equal(t, customStage.Key, dispatch.Steps[1].StageKey)
	assert.Equal(t, "phase-0", dispatch.Steps[2].StageKey)
	require.Len(t, results, 2)
	assert.Equal(t, firstEntry, results[0].Run.NodeID)
	assert.Equal(t, secondEntry, results[1].Run.NodeID)
	assert.Equal(t, []int{0, 1}, []int{results[0].Execution.StepIndex, results[1].Execution.StepIndex})
	var manualRuns int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ? AND node_id = ?", manual.ID, manualEntry).Count(&manualRuns).Error)
	require.Zero(t, manualRuns)
}

func Test__FactoryLine__DispatchStage__EmptyFirstStageUsesAutoAdvance(t *testing.T) {
	for _, test := range []struct {
		name        string
		autoAdvance bool
		rerun       bool
		wantRuns    int
		wantState   string
		wantStage   string
	}{
		{name: "enabled", autoAdvance: true, wantRuns: 1, wantState: models.FactoryWorkOrderLineDispatchStateActive, wantStage: "phase-0"},
		{name: "disabled", autoAdvance: false, wantRuns: 0, wantState: models.FactoryWorkOrderLineDispatchStateFinished, wantStage: "custom-qa"},
		{name: "disabled rerun", autoAdvance: false, rerun: true, wantRuns: 1, wantState: models.FactoryWorkOrderLineDispatchStateActive, wantStage: "phase-0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
			require.NoError(t, err)
			app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
			line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
				Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry,
			}})
			require.NoError(t, err)
			qa := models.FactoryLineStage{
				ID: uuid.New(), Key: "custom-qa", Name: "QA", Kind: models.FactoryLineStagePhase,
				AutoAdvance: &test.autoAdvance,
			}
			stages := append([]models.FactoryLineStage{line.StagesValue()[0], qa}, line.StagesValue()[1:]...)
			require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
			manual, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Manual QA", "manual")
			require.NoError(t, manual.SetColumnKey(db, qa.Key))
			version, err := models.FindLiveCanvasVersionInTransaction(db, manual.ID)
			require.NoError(t, err)
			version.Nodes[0].Metadata = map[string]any{models.FactoryAppTemplateMetadataKey: map[string]any{"startMode": "manual"}}
			require.NoError(t, db.Save(version).Error)
			if test.rerun {
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					_, _, dispatchErr := line.DispatchStage(tx, order)
					return dispatchErr
				}))
			}

			var dispatch *models.FactoryWorkOrderLineDispatch
			var results []*models.FactoryLineStepResult
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				var dispatchErr error
				dispatch, results, dispatchErr = line.DispatchStage(tx, order)
				return dispatchErr
			}))

			assert.Len(t, results, test.wantRuns)
			reloaded, err := models.FindWorkOrderLineDispatch(db, dispatch.ID)
			require.NoError(t, err)
			assert.Equal(t, test.wantState, reloaded.State)
			assert.Equal(t, test.wantStage, reloaded.CurrentStageKey)
		})
	}
}

func Test__FactoryLine__DispatchStage__StopsInEmptyVerifyBeforeFirstRun(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry,
	}})
	require.NoError(t, err)
	stages := line.StagesValue()
	// Backlog, Verify, phase-0, Done.
	stages = []models.FactoryLineStage{stages[0], stages[2], stages[1], stages[3]}
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, dispatchErr := line.DispatchStage(tx, order)
		return dispatchErr
	}))

	inVerify, err := order.IsWaitingAfterPassedStage(db)
	require.NoError(t, err)
	assert.True(t, inVerify)
	column, err := models.WorkOrderBoardColumn(db, order)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasColumnKeyVerify, column)
}

func Test__FactoryWorkOrder__RetryLineStep__KeepsCustomStageSiblingsRunning(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	phase, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: phase.ID, Entrypoint: entrypoint,
	}})
	require.NoError(t, err)
	stageID := uuid.New()
	stage := models.FactoryLineStage{ID: stageID, Key: "custom-" + stageID.String(), Name: "QA", Kind: models.FactoryLineStagePhase}
	stages := append([]models.FactoryLineStage{line.StagesValue()[0], stage}, line.StagesValue()[1:]...)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	first, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Lint", "start")
	second, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Test", "start")
	require.NoError(t, first.SetColumnKey(db, stage.Key))
	require.NoError(t, second.SetColumnKey(db, stage.Key))

	var dispatch *models.FactoryWorkOrderLineDispatch
	var runs []*models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, runs, dispatchErr = line.DispatchStage(tx, order)
		return dispatchErr
	}))
	require.Len(t, runs, 2)
	require.NoError(t, runs[0].Execution.MarkFinished(db, models.CanvasRunResultPassed))

	var later *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var startErr error
		later, startErr = dispatch.EnqueueOrStartStep(tx, order, 2)
		return startErr
	}))
	require.NotNil(t, later.Execution)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, err = order.RetryLineStep(tx, line, 0)
		return err
	}))
	sibling, err := models.FindWorkOrderExecutionByRunID(db, runs[1].Run.ID)
	require.NoError(t, err)
	assert.NotEqual(t, models.FactoryWorkOrderExecutionStatusFinished, sibling.Status)

	settledLater, err := models.FindWorkOrderExecutionByRunID(db, later.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, settledLater.Status)
	assert.Equal(t, models.CanvasRunResultCancelled, settledLater.Result)

	require.NoError(t, dispatch.EnterStage(db, "phase-0"))
	var moved []*models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, moved, err = order.RetryLineStep(tx, line, 1)
		return err
	}))
	require.Len(t, moved, 2, "entering a custom stage starts every automation")
	assert.Equal(t, []int{0, 1}, []int{moved[0].Execution.StepIndex, moved[1].Execution.StepIndex})
}

func Test__FactoryLine__DispatchStage__QueuesCustomStageAboveLimit(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	phase, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: phase.ID, Entrypoint: entrypoint,
	}})
	require.NoError(t, err)
	stageID := uuid.New()
	stage := models.FactoryLineStage{ID: stageID, Key: "custom-" + stageID.String(), Name: "QA", Kind: models.FactoryLineStagePhase}
	stages := append([]models.FactoryLineStage{line.StagesValue()[0], stage}, line.StagesValue()[1:]...)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	lint, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Lint", "start")
	require.NoError(t, lint.SetColumnKey(db, stage.Key))
	tests, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Tests", "start")
	require.NoError(t, tests.SetColumnKey(db, stage.Key))

	limit := models.DefaultFactoryLineStepMaxParallelism
	var lastDispatch *models.FactoryWorkOrderLineDispatch
	var last []*models.FactoryLineStepResult
	for index := 0; index <= limit; index++ {
		order, err := factory.CreateWorkOrder(db, fmt.Sprintf("Order %d", index), "", &r.User, nil, nil)
		require.NoError(t, err)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			var dispatchErr error
			lastDispatch, last, dispatchErr = line.DispatchStage(tx, order)
			return dispatchErr
		}))
	}

	require.Len(t, last, 2)
	for _, result := range last {
		assert.Nil(t, result.Execution)
		require.NotNil(t, result.QueueItem)
		assert.Equal(t, stage.Key, result.QueueItem.StageKey)
	}
	byDispatch, err := models.ListFactoryWorkOrderQueueItemsByLineDispatchIDs(db, []uuid.UUID{lastDispatch.ID})
	require.NoError(t, err)
	assert.Equal(t, min(last[0].QueueItem.StepIndex, last[1].QueueItem.StepIndex), byDispatch[lastDispatch.ID].StepIndex)
}

func Test__FactoryWorkOrder__RetryLineStep__QueuesCustomStageRetryAtLimit(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	phase, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: phase.ID, Entrypoint: entrypoint,
	}})
	require.NoError(t, err)
	stageID := uuid.New()
	stage := models.FactoryLineStage{ID: stageID, Key: "custom-" + stageID.String(), Name: "QA", Kind: models.FactoryLineStagePhase}
	stages := append([]models.FactoryLineStage{line.StagesValue()[0], stage}, line.StagesValue()[1:]...)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	lint, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Lint", "start")
	require.NoError(t, lint.SetColumnKey(db, stage.Key))

	var first *models.FactoryWorkOrder
	for index := 0; index <= models.DefaultFactoryLineStepMaxParallelism; index++ {
		order, err := factory.CreateWorkOrder(db, fmt.Sprintf("Order %d", index), "", &r.User, nil, nil)
		require.NoError(t, err)
		if first == nil {
			first = order
		}
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			if openErr := order.TransitionOnDispatch(tx, &r.User); openErr != nil {
				return openErr
			}
			_, _, dispatchErr := line.DispatchStage(tx, order)
			return dispatchErr
		}))
	}

	var retried *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var retryErr error
		retried, _, retryErr = first.RetryLineStep(tx, line, 0)
		return retryErr
	}))

	assert.Nil(t, retried.Execution)
	require.NotNil(t, retried.QueueItem)
}

func Test__FactoryLine__Queue__KeepsRepeatedAppStageLimitsSeparate(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Repeated", "start")
	one := 1
	ten := 10
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &ten},
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &one},
	})
	require.NoError(t, err)
	first, err := factory.CreateWorkOrder(db, "First", "", &r.User, nil, nil)
	require.NoError(t, err)
	second, err := factory.CreateWorkOrder(db, "Second", "", &r.User, nil, nil)
	require.NoError(t, err)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		firstDispatch, _, dispatchErr := line.Dispatch(tx, first)
		if dispatchErr != nil {
			return dispatchErr
		}
		if _, startErr := firstDispatch.EnqueueOrStartStep(tx, first, 1); startErr != nil {
			return startErr
		}
		secondDispatch, _, dispatchErr := line.Dispatch(tx, second)
		if dispatchErr != nil {
			return dispatchErr
		}
		result, startErr := secondDispatch.EnqueueOrStartStep(tx, second, 1)
		if startErr != nil {
			return startErr
		}
		require.NotNil(t, result.QueueItem)
		return nil
	}))
}

func Test__FactoryLine__Queue__IncludesWorkWithoutStageSnapshots(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	limit := 1
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &limit,
	}})
	require.NoError(t, err)
	first, err := factory.CreateWorkOrder(db, "First", "", &r.User, nil, nil)
	require.NoError(t, err)
	second, err := factory.CreateWorkOrder(db, "Second", "", &r.User, nil, nil)
	require.NoError(t, err)

	var firstRun *models.FactoryLineStepResult
	var secondDispatch *models.FactoryWorkOrderLineDispatch
	var queued *models.FactoryWorkOrderQueueItem
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if openErr := first.TransitionOnDispatch(tx, &r.User); openErr != nil {
			return openErr
		}
		if openErr := second.TransitionOnDispatch(tx, &r.User); openErr != nil {
			return openErr
		}
		firstDispatch, result, dispatchErr := line.Dispatch(tx, first)
		if dispatchErr != nil {
			return dispatchErr
		}
		firstRun = result
		firstDispatch.Steps[0].StageKey = ""
		firstDispatch.Steps[0].StageName = ""
		firstDispatch.Steps[0].StageKind = ""
		if updateErr := tx.Model(firstDispatch).Update("steps", firstDispatch.Steps).Error; updateErr != nil {
			return updateErr
		}

		secondDispatch, result, dispatchErr = line.Dispatch(tx, second)
		if dispatchErr != nil {
			return dispatchErr
		}
		queued = result.QueueItem
		if queued == nil {
			return fmt.Errorf("expected legacy active work to hold the stage slot")
		}
		secondDispatch.Steps[0].StageKey = ""
		secondDispatch.Steps[0].StageName = ""
		secondDispatch.Steps[0].StageKind = ""
		if updateErr := tx.Model(secondDispatch).Update("steps", secondDispatch.Steps).Error; updateErr != nil {
			return updateErr
		}
		return tx.Model(queued).Update("stage_key", "").Error
	}))

	require.NoError(t, firstRun.Execution.MarkFinished(db, models.CanvasRunResultPassed))
	var admitted []*models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var admitErr error
		admitted, admitErr = models.AdmitQueuedForStep(tx, line.ID, app.ID, "phase-0")
		return admitErr
	}))
	require.Len(t, admitted, 1)
	assert.Equal(t, secondDispatch.ID, admitted[0].Execution.LineDispatchID)
}

func Test__FactoryLine__Queue__AdmitsStageWorkWhenWorkWithoutStageSnapshotEnds(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	limit := 1
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &limit,
	}})
	require.NoError(t, err)
	legacy, err := factory.CreateWorkOrder(db, "Legacy", "", &r.User, nil, nil)
	require.NoError(t, err)
	waiting, err := factory.CreateWorkOrder(db, "Waiting", "", &r.User, nil, nil)
	require.NoError(t, err)

	var legacyRun *models.FactoryLineStepResult
	var waitingDispatch *models.FactoryWorkOrderLineDispatch
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		for _, order := range []*models.FactoryWorkOrder{legacy, waiting} {
			if openErr := order.TransitionOnDispatch(tx, &r.User); openErr != nil {
				return openErr
			}
		}
		legacyDispatch, result, dispatchErr := line.Dispatch(tx, legacy)
		if dispatchErr != nil {
			return dispatchErr
		}
		legacyRun = result
		legacyDispatch.Steps[0].StageKey = ""
		legacyDispatch.Steps[0].StageName = ""
		legacyDispatch.Steps[0].StageKind = ""
		if updateErr := tx.Model(legacyDispatch).Update("steps", legacyDispatch.Steps).Error; updateErr != nil {
			return updateErr
		}

		var queued *models.FactoryLineStepResult
		waitingDispatch, queued, dispatchErr = line.Dispatch(tx, waiting)
		if dispatchErr != nil {
			return dispatchErr
		}
		if queued.QueueItem == nil || queued.QueueItem.StageKey != "phase-0" {
			return fmt.Errorf("expected the new task to wait in phase-0")
		}
		return nil
	}))

	require.NoError(t, legacyRun.Execution.MarkFinished(db, models.CanvasRunResultPassed))
	var admitted []*models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var admitErr error
		admitted, admitErr = models.AdmitQueuedForStep(tx, line.ID, app.ID, "")
		return admitErr
	}))
	require.Len(t, admitted, 1)
	assert.Equal(t, waitingDispatch.ID, admitted[0].Execution.LineDispatchID)
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
	}, nil))

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

	_, err = order.AbandonActiveLineDispatch(db)
	require.NoError(t, err)

	_, err = order.FindActiveLineDispatch(db)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__FactoryWorkOrder__Close__CancelsLineRun(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)
	var result *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := order.TransitionOnDispatch(tx, &r.User); err != nil {
			return err
		}
		_, result, err = line.Dispatch(tx, order)
		return err
	}))

	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/42",
	})
	require.NoError(t, err)
	feedbackRun, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(db, feedbackRun.ID, "Review feedback"))
	unrelated, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)

	_, err = order.Close(db, models.FactoryWorkOrderResultRejected, &r.User)
	require.NoError(t, err)
	run, err := models.FindUnscopedCanvasRun(db, result.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, run.State)
	require.NotNil(t, run.CancelledBy)
	assert.Equal(t, r.User, *run.CancelledBy)
	feedbackRun, err = models.FindUnscopedCanvasRun(db, feedbackRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, feedbackRun.State)
	unrelated, err = models.FindUnscopedCanvasRun(db, unrelated.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, unrelated.State)
}

func Test__FactoryWorkOrder__AbandonActiveLineDispatch__FreesTheStepSlot(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "plan", "start-plan")
	limit := 1
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &limit},
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

	var inFlight *models.FactoryWorkOrderExecution
	var queuedDispatch *models.FactoryWorkOrderLineDispatch
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, running)
		if dispatchErr != nil {
			return dispatchErr
		}
		inFlight = result.Execution

		var queuedErr error
		var queuedResult *models.FactoryLineStepResult
		queuedDispatch, queuedResult, queuedErr = line.Dispatch(tx, queued)
		if queuedErr != nil {
			return queuedErr
		}
		if queuedResult.QueueItem == nil {
			return fmt.Errorf("expected the second order to wait for the step slot")
		}
		return nil
	}))

	var started []*models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var abandonErr error
		started, abandonErr = running.AbandonActiveLineDispatch(tx)
		return abandonErr
	}))

	require.NotNil(t, inFlight.RunID)
	settled, err := models.FindWorkOrderExecutionByRunID(db, *inFlight.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, settled.Status)
	assert.Equal(t, models.CanvasRunResultCancelled, settled.Result)

	byDispatch, err := models.ListFactoryWorkOrderQueueItemsByLineDispatchIDs(db, []uuid.UUID{queuedDispatch.ID})
	require.NoError(t, err)
	_, stillQueued := byDispatch[queuedDispatch.ID]
	assert.False(t, stillQueued, "the freed slot must admit the waiting dispatch")

	var admittedRun *models.CanvasRun
	for _, result := range started {
		if result.Execution != nil && result.Execution.WorkOrderID == queued.ID {
			admittedRun = result.Run
		}
	}
	require.NotNil(t, admittedRun, "admitted waiters must be returned so the caller can publish them")
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
	}, nil))

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
	}, nil))

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

	// A finished legacy snapshot must remain usable when a retry reopens it.
	require.NoError(t, db.Exec(`UPDATE factory_work_order_line_dispatches
		SET stages = jsonb_set(stages, '{3,kind}', '"verify"'),
		    steps = jsonb_set(steps, '{1,stage_kind}', '"custom"')
		WHERE id = ?`, full.ID).Error)
	migration, err := os.ReadFile("../../db/migrations/20261124182440_unify-factory-stage-kinds.up.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(migration)).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, dispatchErr := line.DispatchFrom(tx, order, 1)
		if dispatchErr != nil {
			return dispatchErr
		}
		_, abandonErr := order.AbandonActiveLineDispatch(tx)
		return abandonErr
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

func Test__FactoryWorkOrder__RetryLineStep__EndsTheTraversalOnAnotherLine(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Improve AGENTS.md", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "plan", "start-plan")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start-implement")
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}

	ship, err := factory.CreateLine(db, "ship", steps)
	require.NoError(t, err)
	hotfix, err := factory.CreateLine(db, "hotfix", steps)
	require.NoError(t, err)

	var shipDispatch *models.FactoryWorkOrderLineDispatch
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		shipDispatch, _, dispatchErr = ship.Dispatch(tx, order)
		return dispatchErr
	}))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, _, retryErr := order.RetryLineStep(tx, hotfix, 1)
		return retryErr
	}))

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, hotfix.ID, active.LineID, "the rerun opens the traversal on the requested line")

	byOrder, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, []uuid.UUID{order.ID})
	require.NoError(t, err)
	var activeCount int
	for _, record := range byOrder[order.ID] {
		if record.State == models.FactoryWorkOrderLineDispatchStateActive {
			activeCount++
		}
	}
	assert.Equal(t, 1, activeCount, "a work order can only have one active traversal")

	reloaded, err := models.FindWorkOrderLineDispatch(db, shipDispatch.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State)
	assert.Equal(t, models.CanvasRunResultCancelled, reloaded.Result)
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
