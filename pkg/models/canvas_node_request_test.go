package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__LockNodeRequest__OnlyLocksDuePendingRequests(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "schedule-trigger",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
			},
		},
		nil,
	)

	t.Run("locks due pending request", func(t *testing.T) {
		request := createCanvasNodeRequest(t, canvas.ID, "schedule-trigger", models.NodeExecutionRequestStatePending, time.Now().Add(-time.Second))

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			locked, err := models.LockNodeRequest(tx, request.ID)
			require.NoError(t, err)
			assert.Equal(t, request.ID, locked.ID)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("locks due pending request for deleted node", func(t *testing.T) {
		request := createCanvasNodeRequest(t, canvas.ID, "schedule-trigger", models.NodeExecutionRequestStatePending, time.Now().Add(-time.Second))

		var node models.CanvasNode
		require.NoError(t, database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "schedule-trigger").First(&node).Error)
		require.NoError(t, database.Conn().Delete(&node).Error)

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			locked, err := models.LockNodeRequest(tx, request.ID)
			require.NoError(t, err)
			assert.Equal(t, request.ID, locked.ID)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("does not lock completed request", func(t *testing.T) {
		request := createCanvasNodeRequest(t, canvas.ID, "schedule-trigger", models.NodeExecutionRequestStateCompleted, time.Now().Add(-time.Second))

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			_, err := models.LockNodeRequest(tx, request.ID)
			assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("does not lock pending request before run_at", func(t *testing.T) {
		request := createCanvasNodeRequest(t, canvas.ID, "schedule-trigger", models.NodeExecutionRequestStatePending, time.Now().Add(time.Minute))

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			_, err := models.LockNodeRequest(tx, request.ID)
			assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
			return nil
		})
		require.NoError(t, err)
	})
}

func createCanvasNodeRequest(t *testing.T, workflowID uuid.UUID, nodeID, state string, runAt time.Time) *models.CanvasNodeRequest {
	t.Helper()

	request := &models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]any{},
			},
		}),
		State:     state,
		RunAt:     runAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, database.Conn().Create(request).Error)
	return request
}
