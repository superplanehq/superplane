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

func Test__LockExpiredRoutedRootCanvasEventsInTransaction_ReturnsMultipleEligibleEvents(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)
	event1 := createExpiredRootEvent(t, canvas.ID)
	event2 := createExpiredRootEvent(t, canvas.ID)

	var events []models.CanvasEvent
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		events, err = models.LockExpiredRoutedRootCanvasEventsInTransaction(tx, time.Now(), 10)
		return err
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{event1.ID, event2.ID}, canvasEventIDs(events))
}

func Test__LockExpiredRoutedRootCanvasEventsInTransaction_RespectsRetentionWindowBoundary(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)

	expired := createRootEvent(t, canvas.ID)
	updateRootEventAgeAndState(t, expired.ID, 31, models.CanvasEventStateRouted)

	notExpired := createRootEvent(t, canvas.ID)
	updateRootEventAgeAndState(t, notExpired.ID, 29, models.CanvasEventStateRouted)

	var events []models.CanvasEvent
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		events, err = models.LockExpiredRoutedRootCanvasEventsInTransaction(tx, time.Now(), 10)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{expired.ID}, canvasEventIDs(events))
}

func Test__DeleteRootCanvasEventChainsInTransaction_DeletesAllRelatedRows(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)

	root1 := createExpiredRootEvent(t, canvas.ID)
	root2 := createExpiredRootEvent(t, canvas.ID)

	exec1 := createExecution(t, canvas.ID, "component", root1.ID, root1.ID)
	exec2 := createExecution(t, canvas.ID, "component", root2.ID, root2.ID)

	createNodeRequest(t, canvas.ID, "component", exec1.ID, models.NodeExecutionRequestStateCompleted)
	createNodeRequest(t, canvas.ID, "component", exec2.ID, models.NodeExecutionRequestStateCompleted)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.DeleteRootCanvasEventChainsInTransaction(tx, []uuid.UUID{root1.ID, root2.ID})
	}))

	var rootEventCount int64
	require.NoError(t, database.Conn().
		Model(&models.CanvasEvent{}).
		Where("id IN ?", []uuid.UUID{root1.ID, root2.ID}).
		Count(&rootEventCount).Error)
	require.Zero(t, rootEventCount)

	var executionCount int64
	require.NoError(t, database.Conn().
		Model(&models.CanvasNodeExecution{}).
		Where("id IN ?", []uuid.UUID{exec1.ID, exec2.ID}).
		Count(&executionCount).Error)
	require.Zero(t, executionCount)

	var requestCount int64
	require.NoError(t, database.Conn().
		Model(&models.CanvasNodeRequest{}).
		Where("execution_id IN ?", []uuid.UUID{exec1.ID, exec2.ID}).
		Count(&requestCount).Error)
	require.Zero(t, requestCount)
}

func Test__DeleteRootCanvasEventChainsInTransaction_HandlesMoreRootEventsThanChunkSize(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)

	//
	// Create slightly more than one chunk worth of root events to exercise the
	// chunking code path inside DeleteRootCanvasEventChainsInTransaction. The
	// chunk size is large (500), so we rely on the constant being kept in sync;
	// the important property here is that the function correctly handles a
	// multi-chunk input rather than hitting an unbounded `IN (...)` clause.
	//
	const totalRootEvents = 600
	rootEventIDs := make([]uuid.UUID, 0, totalRootEvents)
	for i := 0; i < totalRootEvents; i++ {
		root := createExpiredRootEvent(t, canvas.ID)
		rootEventIDs = append(rootEventIDs, root.ID)
	}

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.DeleteRootCanvasEventChainsInTransaction(tx, rootEventIDs)
	}))

	var remaining int64
	require.NoError(t, database.Conn().
		Model(&models.CanvasEvent{}).
		Where("id IN ?", rootEventIDs).
		Count(&remaining).Error)
	require.Zero(t, remaining)
}

func Test__LockExpiredRoutedRootCanvasEventsInTransaction_ExcludesInactiveAndIneligibleEvents(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := createOrganization(t)
	cacheRetentionWindow(t, org.ID, 30)
	canvas := createRetentionCanvas(t, org.ID)
	eligible := createExpiredRootEvent(t, canvas.ID)

	withinRetention := createRootEvent(t, canvas.ID)
	updateRootEventAgeAndState(t, withinRetention.ID, 29, models.CanvasEventStateRouted)

	noRetentionOrg := createOrganization(t)
	noRetentionCanvas := createRetentionCanvas(t, noRetentionOrg.ID)
	createExpiredRootEvent(t, noRetentionCanvas.ID)

	deletedCanvas := createRetentionCanvas(t, org.ID)
	createExpiredRootEvent(t, deletedCanvas.ID)
	require.NoError(t, database.Conn().Delete(&models.Canvas{}, "id = ?", deletedCanvas.ID).Error)

	deletedOrg := createOrganization(t)
	cacheRetentionWindow(t, deletedOrg.ID, 30)
	deletedOrgCanvas := createRetentionCanvas(t, deletedOrg.ID)
	createExpiredRootEvent(t, deletedOrgCanvas.ID)
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

	var events []models.CanvasEvent
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		events, err = models.LockExpiredRoutedRootCanvasEventsInTransaction(tx, time.Now(), 20)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{eligible.ID}, canvasEventIDs(events))
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
			ID:          versionID,
			WorkflowID:  canvas.ID,
			State:       models.CanvasVersionStatePublished,
			Name:        canvas.Name,
			Description: "test canvas",
			PublishedAt: &now,
			Nodes:       datatypes.NewJSONSlice([]models.Node{}),
			Edges:       datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt:   &now,
			UpdatedAt:   &now,
		}

		return tx.Create(&version).Error
	}))

	return &canvas
}

func createExpiredRootEvent(t *testing.T, canvasID uuid.UUID) *models.CanvasEvent {
	t.Helper()

	event := createRootEvent(t, canvasID)
	updateRootEventAgeAndState(t, event.ID, 31, models.CanvasEventStateRouted)
	return event
}

func createRootEvent(t *testing.T, canvasID uuid.UUID) *models.CanvasEvent {
	t.Helper()

	now := time.Now()
	event := models.CanvasEvent{
		WorkflowID: canvasID,
		NodeID:     "trigger",
		Channel:    "default",
		Data:       datatypes.NewJSONType[any](map[string]any{"key": "value"}),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
	}

	require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&event).Error)
	return &event
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

	now := time.Now()
	queueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  canvasID,
		NodeID:      nodeID,
		RootEventID: rootEventID,
		EventID:     eventID,
		CreatedAt:   &now,
	}

	require.NoError(t, database.Conn().Create(&queueItem).Error)
}

func createExecution(t *testing.T, canvasID uuid.UUID, nodeID string, rootEventID uuid.UUID, eventID uuid.UUID) *models.CanvasNodeExecution {
	t.Helper()

	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    canvasID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		EventID:       eventID,
		State:         models.CanvasNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func canvasEventIDs(events []models.CanvasEvent) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}

	return ids
}
