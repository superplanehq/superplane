package workers

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestHumanReviewRejectsAnUntrackedRunWithoutRetryingOrStartingTheAgent(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}), Configuration: datatypes.NewJSONType(map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"})},
		{NodeID: "agent", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
	}, []models.Edge{{SourceID: models.HumanReviewTriggerNodeID, TargetID: "agent", Channel: "default"}})
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	require.NoError(t, canvas.StampFactoryAppTemplate(db, models.HumanReviewTriggerNodeID, models.FactoryAppTemplateHumanReviewID, 1))
	event := &models.CanvasEvent{WorkflowID: canvas.ID, NodeID: models.HumanReviewTriggerNodeID, Channel: "default", State: models.CanvasEventStatePending, Data: models.NewJSONValue(map[string]any{"type": "factory.humanReview", "data": map[string]any{}})}
	require.NoError(t, db.Create(event).Error)
	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	stored, err := models.FindCanvasEventForCanvas(db, canvas.ID, event.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasEventStateRouted, stored.State)
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, event.RunID)
	require.NoError(t, err)
	require.NotEmpty(t, run.Errors)
	var queued int64
	require.NoError(t, db.Model(&models.CanvasNodeQueueItem{}).Where("run_id = ?", event.RunID).Count(&queued).Error)
	require.Zero(t, queued)
}
