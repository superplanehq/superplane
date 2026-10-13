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

func Test__FactoryWorkOrder__Close__LeavesSharedCreatorRunning(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "intake", "start")
	source, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)

	closing, err := factory.CreateWorkOrder(db, "Closing", "", &r.User, nil, &source.ID)
	require.NoError(t, err)
	_, err = factory.CreateWorkOrder(db, "Other", "", &r.User, nil, &source.ID)
	require.NoError(t, err)

	otherWork, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	ownedWork, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(otherWork).Update("parent_run_id", source.ID).Error)
	require.NoError(t, db.Model(ownedWork).Updates(map[string]any{
		"parent_run_id": source.ID,
		"input": models.NewJSONValue(map[string]any{
			"work_order": map[string]any{"id": closing.ID.String()},
		}),
	}).Error)

	_, err = closing.Close(db, models.FactoryWorkOrderResultRejected, &r.User)
	require.NoError(t, err)

	source, err = models.FindUnscopedCanvasRun(db, source.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, source.State)
	otherWork, err = models.FindUnscopedCanvasRun(db, otherWork.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, otherWork.State)
	ownedWork, err = models.FindUnscopedCanvasRun(db, ownedWork.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, ownedWork.State)
}

func Test__FactoryWorkOrder__Close__CancelsExclusiveCreator(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "intake", "start")
	source, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Only", "", &r.User, nil, &source.ID)
	require.NoError(t, err)

	_, err = order.Close(db, models.FactoryWorkOrderResultRejected, &r.User)
	require.NoError(t, err)

	source, err = models.FindUnscopedCanvasRun(db, source.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, source.State)
}

func Test__FactoryWorkOrder__Close__DoesNotWaitForBusyAnalysisRun(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "analysis", "start")
	run, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Draft", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = factory.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    app.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)

	holding := make(chan struct{})
	release := make(chan struct{})
	holdErr := make(chan error, 1)
	go func() {
		holdErr <- database.Conn().Transaction(func(tx *gorm.DB) error {
			if _, err := models.LockCanvasRunInTransaction(tx, run.ID); err != nil {
				return err
			}
			close(holding)
			select {
			case <-release:
			case <-time.After(2 * time.Second):
			}
			return nil
		})
	}()

	select {
	case <-holding:
	case err := <-holdErr:
		require.NoError(t, err)
		t.Fatal("lock holder ended before close")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting to hold the run lock")
	}

	started := time.Now()
	_, err = order.Close(db, models.FactoryWorkOrderResultArchived, &r.User)
	elapsed := time.Since(started)
	require.NoError(t, err)
	t.Logf("archive close with a busy planning run took %s", elapsed)
	assert.Less(t, elapsed, time.Second)
	assert.True(t, order.RunCancellationDeferred)

	closed, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, closed.State)
	assert.Equal(t, models.FactoryWorkOrderResultArchived, closed.Result)
	session, err := models.FindPlanningSessionByRun(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, session.State)
	busy, err := models.FindUnscopedCanvasRun(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, busy.State)

	close(release)
	require.NoError(t, <-holdErr)

	results, deferred, err := order.FinishDeferredRunCancellations(db, &r.User)
	require.NoError(t, err)
	assert.False(t, deferred)
	require.NotEmpty(t, results)
	assert.True(t, results[0].NewlyCancelling)
	assert.NotNil(t, results[0].Drain)

	cancelled, err := models.FindUnscopedCanvasRun(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, cancelled.State)
}
