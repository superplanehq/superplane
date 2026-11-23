package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestWorkOrderReviewReadyUsesLatestTraversal(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, order.TransitionOnDispatch(db, &r.User))
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Build", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
	require.NoError(t, err)
	dispatch, execution, err := line.RecordPassedStageWithoutRun(db, order, 0)
	require.NoError(t, err)
	assert.Nil(t, execution.RunID)
	assert.True(t, dispatch.ReviewReady)
	column, err := models.WorkOrderBoardColumn(db, order)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasColumnKeyVerify, column)
	require.NoError(t, dispatch.Reactivate(db))
	ready, err := order.IsWaitingAfterPassedStage(db)
	require.NoError(t, err)
	assert.False(t, ready)
	require.NoError(t, dispatch.Finish(db, models.CanvasRunResultPassed))
	later := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, line.Steps)
	require.NoError(t, db.Model(later).Update("created_at", time.Now().Add(time.Second)).Error)
	ready, err = order.IsWaitingAfterPassedStage(db)
	require.NoError(t, err)
	assert.False(t, ready)
	order.State = models.FactoryWorkOrderStateDraft
	column, err = models.WorkOrderBoardColumn(db, order)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasColumnKeyBacklog, column)
}
