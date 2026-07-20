package workers

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
)

func Test__NodeRequestCleanupWorker_DeletesExpiredCompletedRequests(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	expired := createNodeRequestForCleanup(t, canvas.ID, "node-1", models.NodeExecutionRequestStateCompleted, time.Now().AddDate(0, 0, -8))
	recent := createNodeRequestForCleanup(t, canvas.ID, "node-1", models.NodeExecutionRequestStateCompleted, time.Now().AddDate(0, 0, -2))
	pending := createNodeRequestForCleanup(t, canvas.ID, "node-1", models.NodeExecutionRequestStatePending, time.Now().AddDate(0, 0, -8))

	worker := NewNodeRequestCleanupWorker()
	worker.pauseBetweenBatches = 0

	deleted, err := worker.cleanCompletedRequests(time.Now().AddDate(0, 0, -worker.retentionDays), worker.maxDeletesPerTick)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	assert.Equal(t, int64(0), countNodeRequestsByID(t, expired.ID))
	assert.Equal(t, int64(1), countNodeRequestsByID(t, recent.ID))
	assert.Equal(t, int64(1), countNodeRequestsByID(t, pending.ID))
}

func Test__NodeRequestCleanupWorker_RespectsPerTickBudget(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	for i := 0; i < 5; i++ {
		createNodeRequestForCleanup(t, canvas.ID, "node-1", models.NodeExecutionRequestStateCompleted, time.Now().AddDate(0, 0, -10))
	}

	worker := NewNodeRequestCleanupWorker()
	worker.pauseBetweenBatches = 0
	worker.deleteBatchSize = 2
	worker.maxDeletesPerTick = 3

	deleted, err := worker.cleanCompletedRequests(time.Now().AddDate(0, 0, -worker.retentionDays), worker.maxDeletesPerTick)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.CanvasNodeRequest{}).
		Where("workflow_id = ? AND state = ?", canvas.ID, models.NodeExecutionRequestStateCompleted).
		Count(&remaining).Error)
	assert.Equal(t, int64(2), remaining)
}

func createNodeRequestForCleanup(t *testing.T, workflowID uuid.UUID, nodeID, state string, updatedAt time.Time) *models.CanvasNodeRequest {
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
		RunAt:     updatedAt,
		CreatedAt: updatedAt,
		UpdatedAt: updatedAt,
	}
	require.NoError(t, database.Conn().Create(request).Error)

	// GORM may rewrite timestamps on create; force the retention cutoff value.
	require.NoError(t, database.Conn().Model(request).Update("updated_at", updatedAt).Error)
	return request
}

func countNodeRequestsByID(t *testing.T, id uuid.UUID) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.Conn().Model(&models.CanvasNodeRequest{}).
		Where("id = ?", id).
		Count(&count).Error)
	return count
}
