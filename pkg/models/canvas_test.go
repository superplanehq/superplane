package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__Canvas__DeleteRemainingResources__ReturnsCompleteWhenUnderLimit(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Type: models.NodeTypeTrigger},
		},
		nil,
	)

	createOrphanNodeRequests(t, canvas.ID, "node-1", 5)

	summary, complete, err := canvas.DeleteRemainingResources(database.Conn(), 500)
	require.NoError(t, err)
	require.True(t, complete)
	require.Equal(t, &models.RunDeletionSummary{NodeRequests: 5}, summary)
	support.VerifyNodeRequestCount(t, canvas.ID, 0)
}

func Test__Canvas__DeleteRemainingResources__StopsBeforeLaterResourceTypesWhenLimitReached(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Type: models.NodeTypeTrigger},
		},
		nil,
	)

	createOrphanNodeRequests(t, canvas.ID, "node-1", 300)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)

	summary, complete, err := canvas.DeleteRemainingResources(database.Conn(), 250)
	require.NoError(t, err)
	require.False(t, complete)
	require.Equal(t, &models.RunDeletionSummary{NodeRequests: 250}, summary)
	support.VerifyNodeRequestCount(t, canvas.ID, 50)
	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
}

func Test__Canvas__DeleteRemainingResources__DeletesRemainingRowsOnNextCall(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Type: models.NodeTypeTrigger},
		},
		nil,
	)

	createOrphanNodeRequests(t, canvas.ID, "node-1", 300)

	summary, complete, err := canvas.DeleteRemainingResources(database.Conn(), 250)
	require.NoError(t, err)
	require.False(t, complete)
	require.Equal(t, int64(250), summary.NodeRequests)

	summary, complete, err = canvas.DeleteRemainingResources(database.Conn(), 250)
	require.NoError(t, err)
	require.True(t, complete)
	require.Equal(t, &models.RunDeletionSummary{NodeRequests: 50}, summary)
	support.VerifyNodeRequestCount(t, canvas.ID, 0)
}

func Test__Canvas__DeleteRemainingResources__ReturnsIncompleteWhenFirstTypeExceedsLimit(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Type: models.NodeTypeTrigger},
		},
		nil,
	)

	createOrphanNodeRequests(t, canvas.ID, "node-1", 600)

	summary, complete, err := canvas.DeleteRemainingResources(database.Conn(), 500)
	require.NoError(t, err)
	require.False(t, complete)
	require.Equal(t, &models.RunDeletionSummary{NodeRequests: 500}, summary)
	support.VerifyNodeRequestCount(t, canvas.ID, 100)
}

func Test__Canvas__DeleteRemainingResources__ReturnsIncompleteWhenLastTypeExceedsLimit(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Type: models.NodeTypeTrigger},
		},
		nil,
	)

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)

	for range 600 {
		run := models.CanvasRun{
			ID:         uuid.New(),
			WorkflowID: canvas.ID,
			NodeID:     "node-1",
			VersionID:  liveVersion.ID,
			State:      models.CanvasRunStateFinished,
			Result:     models.CanvasRunResultPassed,
			FinishedAt: &now,
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}
		require.NoError(t, database.Conn().Create(&run).Error)
	}

	summary, complete, err := canvas.DeleteRemainingResources(database.Conn(), 500)
	require.NoError(t, err)
	require.False(t, complete)
	require.Equal(t, &models.RunDeletionSummary{Runs: 500}, summary)

	var remainingRuns int64
	require.NoError(t, database.Conn().Model(&models.CanvasRun{}).Where("workflow_id = ?", canvas.ID).Count(&remainingRuns).Error)
	require.Equal(t, int64(100), remainingRuns)
}

func Test__AvailableCanvasName__LooksOnlyAtTheOwningWorkspace(t *testing.T) {
	r := support.Setup(t)

	workspace := createWorkspace(t, r.Organization.ID)
	otherWorkspace := createWorkspace(t, r.Organization.ID)
	createCanvasNamed(t, r.Organization.ID, r.User, &workspace.ID, "Plan")

	t.Run("steps aside for a name the workspace already holds", func(t *testing.T) {
		name, err := models.AvailableCanvasName(database.Conn(), r.Organization.ID, &workspace.ID, "Plan")
		require.NoError(t, err)
		require.Equal(t, "Plan (2)", name)
	})

	t.Run("keeps the name in another workspace", func(t *testing.T) {
		name, err := models.AvailableCanvasName(database.Conn(), r.Organization.ID, &otherWorkspace.ID, "Plan")
		require.NoError(t, err)
		require.Equal(t, "Plan", name)
	})

	t.Run("keeps the name outside every workspace", func(t *testing.T) {
		name, err := models.AvailableCanvasName(database.Conn(), r.Organization.ID, nil, "Plan")
		require.NoError(t, err)
		require.Equal(t, "Plan", name)
	})
}

func Test__FindCanvasByName__LooksOnlyAtTheOwningWorkspace(t *testing.T) {
	r := support.Setup(t)

	workspace := createWorkspace(t, r.Organization.ID)
	otherWorkspace := createWorkspace(t, r.Organization.ID)
	canvas := createCanvasNamed(t, r.Organization.ID, r.User, &workspace.ID, "Plan")

	t.Run("finds the app in its own workspace", func(t *testing.T) {
		found, err := models.FindCanvasByName(database.Conn(), r.Organization.ID, &workspace.ID, "Plan")
		require.NoError(t, err)
		require.Equal(t, canvas.ID, found.ID)
	})

	t.Run("does not reach into another workspace", func(t *testing.T) {
		_, err := models.FindCanvasByName(database.Conn(), r.Organization.ID, &otherWorkspace.ID, "Plan")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("does not reach into a workspace from outside", func(t *testing.T) {
		_, err := models.FindCanvasByName(database.Conn(), r.Organization.ID, nil, "Plan")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func createWorkspace(t *testing.T, organizationID uuid.UUID) *models.Factory {
	t.Helper()

	workspace, err := models.CreateFactory(database.Conn(), organizationID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	return workspace
}

func createCanvasNamed(t *testing.T, organizationID, userID uuid.UUID, factoryID *uuid.UUID, name string) *models.Canvas {
	t.Helper()

	canvas, _ := support.CreateCanvas(t, organizationID, userID, nil, nil)
	require.NoError(t, database.Conn().Model(canvas).Updates(map[string]any{
		"name":       name,
		"factory_id": factoryID,
	}).Error)
	canvas.Name = name
	canvas.FactoryID = factoryID

	return canvas
}

func createOrphanNodeRequests(t *testing.T, workflowID uuid.UUID, nodeID string, count int) {
	t.Helper()

	now := time.Now()
	for range count {
		request := models.CanvasNodeRequest{
			ID:         uuid.New(),
			WorkflowID: workflowID,
			NodeID:     nodeID,
			Type:       models.NodeRequestTypeInvokeAction,
			State:      models.NodeExecutionRequestStatePending,
			Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
				InvokeAction: &models.InvokeAction{
					ActionName: "test",
					Parameters: map[string]any{},
				},
			}),
			RunAt:     now,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, database.Conn().Create(&request).Error)
	}
}
