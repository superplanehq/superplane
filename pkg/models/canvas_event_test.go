package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Test__LockRetainedFinishedRuns_ReturnsMultipleEligibleRuns(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)
	event1 := createExpiredRootEvent(t, canvas.ID)
	event2 := createExpiredRootEvent(t, canvas.ID)

	var runs []models.CanvasRun
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		runs, err = models.LockRetainedFinishedRuns(tx, time.Now(), 10)
		return err
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{
		runIDForRootEvent(t, event1.ID),
		runIDForRootEvent(t, event2.ID),
	}, canvasRunIDs(runs))
}

func Test__LockRetainedFinishedRuns_IncludesSoftDeletedCanvasesAndOrganizations(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)
	eligible := createExpiredRootEvent(t, canvas.ID)

	withinRetention := createRootEvent(t, canvas.ID)
	updateRootEventAgeAndState(t, withinRetention.ID, 29, models.CanvasEventStateRouted)
	markRunFinishedForRootEvent(t, withinRetention.ID, 29)

	noRetentionOrg := createOrganization(t)
	noRetentionCanvas := createRetentionCanvas(t, noRetentionOrg.ID)
	createExpiredRootEvent(t, noRetentionCanvas.ID)

	deletedCanvas := createRetentionCanvas(t, org.ID)
	deletedCanvasEvent := createExpiredRootEvent(t, deletedCanvas.ID)
	require.NoError(t, database.Conn().Delete(&models.Canvas{}, "id = ?", deletedCanvas.ID).Error)

	deletedOrg := createOrganization(t)
	cacheRetentionWindow(t, deletedOrg.ID, 30)
	deletedOrgCanvas := createRetentionCanvas(t, deletedOrg.ID)
	deletedOrgEvent := createExpiredRootEvent(t, deletedOrgCanvas.ID)
	require.NoError(t, database.Conn().Delete(&models.Organization{}, "id = ?", deletedOrg.ID).Error)

	queuedRoot := createExpiredRootEvent(t, canvas.ID)
	createQueueItem(t, canvas.ID, "component", queuedRoot.ID, queuedRoot.ID)

	activeRoot := createExpiredRootEvent(t, canvas.ID)
	activeExecution := createExecution(t, canvas.ID, "component", activeRoot.ID, activeRoot.ID)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNodeExecution{}).
		Where("id = ?", activeExecution.ID).
		Update("state", models.CanvasNodeExecutionStateStarted).
		Error)

	pendingRequestRoot := createExpiredRootEvent(t, canvas.ID)
	pendingRequestExecution := createExecution(t, canvas.ID, "component", pendingRequestRoot.ID, pendingRequestRoot.ID)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNodeExecution{}).
		Where("id = ?", pendingRequestExecution.ID).
		Updates(map[string]any{
			"state":  models.CanvasNodeExecutionStateFinished,
			"result": models.CanvasNodeExecutionResultPassed,
		}).
		Error)
	createNodeRequest(t, canvas.ID, "component", pendingRequestExecution.ID, models.NodeExecutionRequestStatePending)

	var runs []models.CanvasRun
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		runs, err = models.LockRetainedFinishedRuns(tx, time.Now(), 20)
		return err
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{
		runIDForRootEvent(t, eligible.ID),
		runIDForRootEvent(t, deletedCanvasEvent.ID),
		runIDForRootEvent(t, deletedOrgEvent.ID),
	}, canvasRunIDs(runs))
}

func Test__LockRetainedFinishedRuns_ExcludesOpenWorkByRunIDWhenRootEventIDIsNull(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)

	eligible := createExpiredRootEvent(t, canvas.ID)

	queuedRoot := createExpiredRootEvent(t, canvas.ID)
	queuedRunID := runIDForRootEvent(t, queuedRoot.ID)
	createQueueItem(t, canvas.ID, "component", queuedRoot.ID, queuedRoot.ID)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNodeQueueItem{}).
		Where("run_id = ?", queuedRunID).
		Update("root_event_id", nil).
		Error)

	activeRoot := createExpiredRootEvent(t, canvas.ID)
	activeExecution := createExecution(t, canvas.ID, "component", activeRoot.ID, activeRoot.ID)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNodeExecution{}).
		Where("id = ?", activeExecution.ID).
		Updates(map[string]any{
			"state":         models.CanvasNodeExecutionStateStarted,
			"root_event_id": nil,
		}).
		Error)

	pendingRequestRoot := createExpiredRootEvent(t, canvas.ID)
	pendingRequestExecution := createExecution(t, canvas.ID, "component", pendingRequestRoot.ID, pendingRequestRoot.ID)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNodeExecution{}).
		Where("id = ?", pendingRequestExecution.ID).
		Updates(map[string]any{
			"state":         models.CanvasNodeExecutionStateFinished,
			"result":        models.CanvasNodeExecutionResultPassed,
			"root_event_id": nil,
		}).
		Error)
	createNodeRequest(t, canvas.ID, "component", pendingRequestExecution.ID, models.NodeExecutionRequestStatePending)

	var runs []models.CanvasRun
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		runs, err = models.LockRetainedFinishedRuns(tx, time.Now(), 20)
		return err
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{runIDForRootEvent(t, eligible.ID)}, canvasRunIDs(runs))
}

func createOrganization(t *testing.T) *models.Organization {
	t.Helper()

	org, err := models.CreateOrganization("org-"+uuid.NewString(), "test org")
	require.NoError(t, err)
	return org
}

func createRetentionCanvas(t *testing.T, orgID uuid.UUID) *models.Canvas {
	t.Helper()

	now := time.Now()
	versionID := uuid.New()
	canvas := models.Canvas{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LiveVersionID:  &versionID,
		Name:           "canvas-" + uuid.NewString(),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&canvas).Error; err != nil {
			return err
		}

		nodes := []models.CanvasNode{
			{
				WorkflowID: canvas.ID,
				NodeID:     "trigger",
				Type:       models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "start"},
				}),
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			{
				WorkflowID: canvas.ID,
				NodeID:     "component",
				Type:       models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
				CreatedAt: &now,
				UpdatedAt: &now,
			},
		}

		if err := tx.Create(&nodes).Error; err != nil {
			return err
		}

		version := models.CanvasVersion{
			ID:         versionID,
			WorkflowID: canvas.ID,
			Nodes:      datatypes.NewJSONSlice([]models.Node{}),
			Edges:      datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}

		return tx.Create(&version).Error
	}))

	return &canvas
}

func createExpiredRootEvent(t *testing.T, canvasID uuid.UUID) *models.CanvasEvent {
	t.Helper()

	event := createRootEvent(t, canvasID)
	updateRootEventAgeAndState(t, event.ID, 31, models.CanvasEventStateRouted)
	markRunFinishedForRootEvent(t, event.ID, 31)
	return event
}

func createRootEvent(t *testing.T, canvasID uuid.UUID) *models.CanvasEvent {
	t.Helper()

	now := time.Now()
	event := models.CanvasEvent{
		WorkflowID: canvasID,
		NodeID:     "trigger",
		Channel:    "default",
		Data:       models.NewJSONValue(map[string]any{"key": "value"}),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
	}

	require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&event).Error)
	return &event
}

func markRunFinishedForRootEvent(t *testing.T, rootEventID uuid.UUID, daysAgo int) {
	t.Helper()

	run, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), rootEventID)
	require.NoError(t, err)

	finishedAt := time.Now().AddDate(0, 0, -daysAgo)
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      models.CanvasRunResultPassed,
		"finished_at": finishedAt,
		"updated_at":  finishedAt,
	}).Error)
}

func runIDForRootEvent(t *testing.T, rootEventID uuid.UUID) uuid.UUID {
	t.Helper()

	run, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), rootEventID)
	require.NoError(t, err)
	return run.ID
}

func updateRootEventAgeAndState(t *testing.T, id uuid.UUID, daysAgo int, state string) {
	t.Helper()

	require.NoError(t, database.Conn().
		Model(&models.CanvasEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"state":      state,
			"created_at": time.Now().AddDate(0, 0, -daysAgo),
		}).
		Error)
}

func cacheRetentionWindow(t *testing.T, orgID uuid.UUID, retentionWindowDays int32) {
	t.Helper()

	require.NoError(t, models.MarkOrganizationUsageLimitsSynced(orgID.String(), &retentionWindowDays, time.Now()))
}

func createNodeRequest(t *testing.T, canvasID uuid.UUID, nodeID string, executionID uuid.UUID, state string) {
	t.Helper()

	now := time.Now()
	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvasID,
		NodeID:      nodeID,
		ExecutionID: &executionID,
		State:       state,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{ActionName: "test", Parameters: map[string]any{}},
		}),
		RunAt:     now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	require.NoError(t, database.Conn().Create(&request).Error)
}

func createQueueItem(t *testing.T, canvasID uuid.UUID, nodeID string, rootEventID uuid.UUID, eventID uuid.UUID) {
	t.Helper()

	run, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), rootEventID)
	require.NoError(t, err)

	now := time.Now()
	queueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvasID,
		NodeID:      nodeID,
		RootEventID: rootEventID,
		RunID:       run.ID,
		EventID:     eventID,
		CreatedAt:   &now,
	}

	require.NoError(t, database.Conn().Create(&queueItem).Error)
}

func createExecution(t *testing.T, canvasID uuid.UUID, nodeID string, rootEventID uuid.UUID, eventID uuid.UUID) *models.CanvasNodeExecution {
	t.Helper()

	run, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), rootEventID)
	require.NoError(t, err)

	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    canvasID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		RunID:         run.ID,
		EventID:       eventID,
		State:         models.CanvasNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func canvasRunIDs(runs []models.CanvasRun) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.ID)
	}

	return ids
}
