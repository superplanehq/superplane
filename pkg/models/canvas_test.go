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
