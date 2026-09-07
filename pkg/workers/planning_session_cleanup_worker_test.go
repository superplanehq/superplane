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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__PlanningSessionCleanupWorker_EndsStaleSessions(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entrypoint := createPlanningCleanupCanvas(t, db, r, factoryModel.ID)
	session, err := factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(session).Update("heartbeat_at", time.Now().Add(-models.PlanningSessionHeartbeatStale-time.Minute)).Error)

	worker := NewPlanningSessionCleanupWorker()
	worker.tick(context.Background())

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, reloaded.State)
}

func createPlanningCleanupCanvas(
	t *testing.T,
	db *gorm.DB,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
) (*models.Canvas, string) {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	entrypoint := "start"
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           support.RandomName("planning-cleanup"),
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		node := models.CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     entrypoint,
			Name:       "Planning",
			Type:       models.NodeTypeTrigger,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onRun"},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		version := models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			Nodes: datatypes.NewJSONSlice([]models.Node{{
				ID:   entrypoint,
				Name: "Planning",
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}},
			}}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas, entrypoint
}
