package models_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
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

func Test__FactoryWorkOrder__Close__CancelsEventAutomationsAndDescendants(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Archive", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "automation", "start")
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other"), "", "")
	require.NoError(t, err)
	childApp, childEntrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "child", "start")
	otherApp, otherEntrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "other", "start")
	unrelated, err := models.CreateCanvasRunInTransaction(db, otherApp.ID, otherEntrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(unrelated).Update("input", models.NewJSONValue(map[string]any{
		"work_order": map[string]any{"id": order.ID.String()},
	})).Error)

	var runIDs []uuid.UUID
	for _, key := range []string{"workOrder", "work_order"} {
		for _, wrapped := range []bool{false, true} {
			run, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			payload := map[string]any{key: map[string]any{"id": order.ID.String()}}
			if wrapped {
				payload = map[string]any{"data": payload}
			}
			require.NoError(t, db.Create(&models.CanvasEvent{
				WorkflowID: app.ID, NodeID: entrypoint, RunID: run.ID,
				State: models.CanvasEventStatePending, Data: models.NewJSONValue(payload),
			}).Error)
			runIDs = append(runIDs, run.ID)
		}
	}
	// Finished parents must still lead to active descendants in other apps.
	finished, err := models.CreateCanvasRunInTransaction(db, childApp.ID, childEntrypoint, models.CanvasRunStateFinished, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(finished).Update("parent_run_id", runIDs[0]).Error)
	descendant, err := models.CreateCanvasRunInTransaction(db, childApp.ID, childEntrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(descendant).Update("parent_run_id", finished.ID).Error)
	runIDs = append(runIDs, descendant.ID)

	_, err = order.Close(db, models.FactoryWorkOrderResultArchived, &r.User)
	require.NoError(t, err)
	for _, id := range runIDs {
		assertRunState(t, db, id, models.CanvasRunStateCancelling)
	}
	assertRunState(t, db, finished.ID, models.CanvasRunStateFinished)
	assertRunState(t, db, unrelated.ID, models.CanvasRunStateStarted)
}

func Test__FactoryWorkOrder__ListRuns__SavesLongIDsAndFindsTaskUUID(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "automation", "start")
	order, err := factoryModel.CreateWorkOrder(db, "Archive", "", &r.User, nil, nil)
	require.NoError(t, err)

	longID := incompressibleText(t, 3000)
	longRun, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(longRun).Update("input", models.NewJSONValue(map[string]any{
		"work_order": map[string]any{"id": longID},
	})).Error)
	for _, payload := range []map[string]any{
		{
			"work_order": map[string]any{"id": longID},
			"workOrder":  map[string]any{"id": longID},
		},
		{"data": map[string]any{
			"work_order": map[string]any{"id": longID},
			"workOrder":  map[string]any{"id": longID},
		}},
	} {
		require.NoError(t, db.Create(&models.CanvasEvent{
			WorkflowID: app.ID, NodeID: entrypoint, RunID: longRun.ID,
			State: models.CanvasEventStatePending, Data: models.NewJSONValue(payload),
		}).Error)
	}

	matched, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.CanvasEvent{
		WorkflowID: app.ID, NodeID: entrypoint, RunID: matched.ID,
		State: models.CanvasEventStatePending,
		Data:  models.NewJSONValue(map[string]any{"work_order": map[string]any{"id": order.ID.String()}}),
	}).Error)

	runs, err := order.ListRuns(db)
	require.NoError(t, err)
	ids := make([]uuid.UUID, len(runs))
	for i := range runs {
		ids[i] = runs[i].ID
	}
	assert.Contains(t, ids, matched.ID)
	assert.NotContains(t, ids, longRun.ID)
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

	results, remaining, err := models.FinishDeferredRunCancellations(db, &r.User, order.DeferredRunCancellationIDs, uuid.Nil)
	require.NoError(t, err)
	assert.Empty(t, remaining)
	require.NotEmpty(t, results)
	assert.True(t, results[0].NewlyCancelling)
	assert.NotNil(t, results[0].Drain)

	cancelled, err := models.FindUnscopedCanvasRun(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, cancelled.State)
}

func Test__FactoryWorkOrder__FinishDeferredRunCancellations__KeepsExcludedAndLaterRuns(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, planning, closer, other := closedOrderWithLockedRuns(t, db, r)
	order, err := factoryModel.FindWorkOrder(db, planning.orderID)
	require.NoError(t, err)

	release := holdCanvasRunLocks(t, planning.runID, closer.ID, other.ID)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultArchived,
		Actor:   &r.User,
		Run:     &factory.RunRef{ID: closer.ID},
	})
	require.NoError(t, err)
	assert.NotContains(t, order.DeferredRunCancellationIDs, closer.ID)
	assert.Contains(t, order.DeferredRunCancellationIDs, planning.runID)
	assert.Contains(t, order.DeferredRunCancellationIDs, other.ID)
	release()

	later, err := models.CreateCanvasRunInTransaction(db, planning.workflowID, planning.nodeID, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(later).Update("input", models.NewJSONValue(map[string]any{
		"work_order": map[string]any{"id": order.ID.String()},
	})).Error)

	ids := append(append([]uuid.UUID{}, order.DeferredRunCancellationIDs...), closer.ID)
	_, remaining, err := models.FinishDeferredRunCancellations(db, &r.User, ids, closer.ID)
	require.NoError(t, err)
	assert.Empty(t, remaining)

	assertRunState(t, db, planning.runID, models.CanvasRunStateCancelling)
	assertRunState(t, db, other.ID, models.CanvasRunStateCancelling)
	assertRunState(t, db, closer.ID, models.CanvasRunStateStarted)
	assertRunState(t, db, later.ID, models.CanvasRunStateStarted)
}

func Test__StopRunForClosedPlanningOrder__StopsDescendantAndLeavesClosingRun(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, planning, closer, other := closedOrderWithLockedRuns(t, db, r)
	order, err := factoryModel.FindWorkOrder(db, planning.orderID)
	require.NoError(t, err)

	release := holdCanvasRunLocks(t, planning.runID, closer.ID, other.ID)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultArchived,
		Actor:   &r.User,
		Run:     &factory.RunRef{ID: closer.ID},
	})
	require.NoError(t, err)
	release()

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		stopped, err := models.StopRunForClosedPlanningOrder(tx, &models.CanvasRun{ID: other.ID, State: models.CanvasRunStateStarted})
		if err != nil {
			return err
		}
		require.NotNil(t, stopped)
		require.NotNil(t, stopped.Drain)
		leftRunning, err := models.StopRunForClosedPlanningOrder(tx, &models.CanvasRun{ID: closer.ID, State: models.CanvasRunStateStarted})
		if err != nil {
			return err
		}
		assert.Nil(t, leftRunning)
		return nil
	}))

	assertRunState(t, db, other.ID, models.CanvasRunStateCancelling)
	assertRunState(t, db, closer.ID, models.CanvasRunStateStarted)
}

func Test__StopRunForClosedPlanningOrder__CancelsRunTheCallerAlreadyLocked(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, planning, _, other := closedOrderWithLockedRuns(t, db, r)
	order, err := factoryModel.FindWorkOrder(db, planning.orderID)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultArchived,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", other.ID).Update("state", models.CanvasRunStateStarted).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockCanvasRunInTransaction(tx, other.ID); err != nil {
			return err
		}
		stopped, err := models.StopRunForClosedPlanningOrder(tx, &models.CanvasRun{ID: other.ID})
		if err != nil {
			return err
		}
		require.NotNil(t, stopped)
		require.NotNil(t, stopped.Drain)
		return nil
	}))
	assertRunState(t, db, other.ID, models.CanvasRunStateCancelling)
}

type lockedPlanningRuns struct {
	orderID    uuid.UUID
	runID      uuid.UUID
	workflowID uuid.UUID
	nodeID     string
}

func closedOrderWithLockedRuns(t *testing.T, db *gorm.DB, r *support.ResourceRegistry) (*models.Factory, lockedPlanningRuns, *models.CanvasRun, *models.CanvasRun) {
	t.Helper()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "analysis", "start")
	planning, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	closer, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	other, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(closer).Update("parent_run_id", planning.ID).Error)
	require.NoError(t, db.Model(other).Update("parent_run_id", planning.ID).Error)
	order, err := factoryModel.CreateWorkOrder(db, "Draft", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    app.ID,
		CanvasRunID: planning.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	return factoryModel, lockedPlanningRuns{
		orderID:    order.ID,
		runID:      planning.ID,
		workflowID: app.ID,
		nodeID:     entrypoint,
	}, closer, other
}

func holdCanvasRunLocks(t *testing.T, runIDs ...uuid.UUID) func() {
	t.Helper()
	holding := make(chan struct{})
	release := make(chan struct{})
	holdErr := make(chan error, 1)
	go func() {
		holdErr <- database.Conn().Transaction(func(tx *gorm.DB) error {
			ids := append([]uuid.UUID(nil), runIDs...)
			slices.SortFunc(ids, func(a, b uuid.UUID) int {
				return bytes.Compare(a[:], b[:])
			})
			for _, id := range ids {
				if _, err := models.LockCanvasRunInTransaction(tx, id); err != nil {
					return err
				}
			}
			close(holding)
			select {
			case <-release:
			case <-time.After(5 * time.Second):
			}
			return nil
		})
	}()
	select {
	case <-holding:
	case err := <-holdErr:
		require.NoError(t, err)
		t.Fatal("lock holder ended before the test")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting to hold the run lock")
	}
	return func() {
		t.Helper()
		close(release)
		require.NoError(t, <-holdErr)
	}
}

func incompressibleText(t *testing.T, size int) string {
	t.Helper()
	raw := make([]byte, size)
	_, err := rand.Read(raw)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func assertRunState(t *testing.T, db *gorm.DB, runID uuid.UUID, state string) {
	t.Helper()
	run, err := models.FindUnscopedCanvasRun(db, runID)
	require.NoError(t, err)
	assert.Equal(t, state, run.State)
}
