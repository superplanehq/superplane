package workers

import (
	"context"
	"testing"
	"time"

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
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning-cleanup", "start")
	session, err := factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(session).Update("heartbeat_at", time.Now().Add(-2*time.Minute)).Error)

	worker := NewPlanningSessionCleanupWorker()
	worker.tick(context.Background())

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, reloaded.State)
}
