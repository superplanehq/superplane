package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestWorkOrderBoardColumnFollowsRetriedAttempt(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Retry implementation", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, order.TransitionOnDispatch(db, &r.User))
	implement, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
	custom, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "custom", "start")
	stageID := uuid.New()
	stageKey := "custom-" + stageID.String()
	require.NoError(t, custom.SetColumnKey(db, stageKey))
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: implement.ID, Entrypoint: entry},
	})
	require.NoError(t, err)
	stages := line.StagesValue()
	stages = append(stages[:2], append([]models.FactoryLineStage{{ID: stageID, Key: stageKey, Name: "QA", Kind: models.FactoryLineStageCustom}}, stages[2:]...)...)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	dispatch, first, err := line.Dispatch(db, order)
	require.NoError(t, err)
	require.NoError(t, first.Execution.MarkFinished(db, models.CanvasRunResultPassed))
	later, err := dispatch.EnqueueOrStartStep(db, order, 1)
	require.NoError(t, err)
	var retry *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		retry, _, err = order.RetryLineStep(tx, line, 0)
		return err
	}))
	column, err := models.WorkOrderBoardColumn(db, order)
	require.NoError(t, err)
	require.Equal(t, models.CanvasColumnKeyImplement, column)
	require.NoError(t, retry.Execution.MarkFinished(db, models.CanvasRunResultFailed))
	column, err = models.WorkOrderBoardColumn(db, order)
	require.NoError(t, err)
	require.Equal(t, models.CanvasColumnKeyImplement, column, "cancelled later work does not own a failed retry")
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		retry, _, err = order.RetryLineStep(tx, line, 0)
		return err
	}))
	require.NoError(t, retry.Execution.MarkFinished(db, models.CanvasRunResultPassed))
	column, err = models.WorkOrderBoardColumn(db, order)
	require.NoError(t, err)
	require.Equal(t, models.CanvasColumnKeyVerify, column)
	waiting, err := order.IsWaitingAfterPassedStage(db)
	require.NoError(t, err)
	require.True(t, waiting)
	updated, err := models.FindWorkOrderExecutionByRunID(db, later.Run.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasRunResultCancelled, updated.Result)
}
