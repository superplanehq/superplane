package support

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// CreateCodexOwnedRun supplies a persisted task association for a subscription
// runner. Callers exercise reservation, callbacks, or runner delivery with it.
func CreateCodexOwnedRun(t *testing.T, db *gorm.DB, connection *models.FactoryCodexConnection, runnerTaskIDs ...uuid.UUID) uuid.UUID {
	t.Helper()
	return CreateCodexOwnedRunInWorkspace(t, db, connection, CodexWorkspaceID(t, db, connection.OrganizationID), runnerTaskIDs...)
}

func CreateCodexOwnedRunInWorkspace(t *testing.T, db *gorm.DB, connection *models.FactoryCodexConnection, factoryID uuid.UUID, runnerTaskIDs ...uuid.UUID) uuid.UUID {
	t.Helper()
	canvas, _ := CreateCanvas(t, connection.OrganizationID, connection.OwnerUserID, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryID).Error)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	factory, err := models.FindFactory(db, connection.OrganizationID, factoryID)
	require.NoError(t, err)
	_, err = factory.CreateWorkOrder(db, "Run subscription task", "", &connection.OwnerUserID, []uuid.UUID{connection.OwnerUserID}, &run.ID)
	require.NoError(t, err)
	for _, taskID := range runnerTaskIDs {
		event := EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		execution := CreateCanvasNodeExecution(t, canvas.ID, "trigger", event.ID, event.ID)
		require.NoError(t, db.Model(execution).Update("run_id", run.ID).Error)
		require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, "trigger", execution.ID, "task_id", taskID.String()))
	}
	return run.ID
}

// CodexWorkspaceID resolves the single workspace created by a subscription
// fixture. Scenarios with multiple workspaces use their explicit workspace IDs.
func CodexWorkspaceID(t *testing.T, db *gorm.DB, organizationID uuid.UUID) uuid.UUID {
	t.Helper()
	var factory models.Factory
	require.NoError(t, db.Where("organization_id = ?", organizationID).Order("created_at, id").First(&factory).Error)
	return factory.ID
}
