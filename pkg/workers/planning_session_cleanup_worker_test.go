package workers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__PlanningSessionCleanupWorker_EndsStaleSessions(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, entrypoint, models.CanvasRunStatePending, "")
	require.NoError(t, err)
	now := time.Now()
	session := &models.FactoryPlanningSession{
		ID:              uuid.New(),
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		CreatedByUserID: &r.User,
		Repository:      "acme/payments",
		Kind:            models.PlanningSessionKindTaskCreation,
		State:           models.PlanningSessionStateRunning,
		CanvasID:        &canvas.ID,
		CanvasRunID:     &run.ID,
		HeartbeatAt:     now.Add(-models.PlanningSessionHeartbeatStale - time.Minute),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, db.Create(session).Error)

	worker := NewPlanningSessionCleanupWorker()
	worker.tick(context.Background())

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, reloaded.State)

	reloadedRun, err := models.FindCanvasRunInTransaction(db, canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, reloadedRun.State)
}
