package workers

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestTaskRootAdmissionWaitsForAnalysisCancellationWithSpareCapacity(t *testing.T) {
	r := support.Setup(t)
	setRunnerPlanLimit(t, r, 2)
	event, order := createBacklogAdmissionEvent(t, r)
	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	db := database.DB(t.Context())
	run, err := models.FindUnscopedCanvasRun(db, event.RunID)
	require.NoError(t, err)
	child, err := models.CreateCanvasRunInTransaction(db, run.WorkflowID, run.NodeID, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(child).Update("parent_run_id", run.ID).Error)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := models.LockOrganizationTaskAdmission(tx, r.Organization.ID); err != nil {
			return err
		}
		_, err := run.RequestCancellation(tx, nil)
		return err
	}))
	replacement := support.EmitCanvasEventForNodeWithData(t, event.WorkflowID, event.NodeID, event.Channel, nil, map[string]any{
		"type": "workOrder.created",
		"data": map[string]any{
			models.WorkOrderCreatedRefinementEnabledDataKey: true,
			"workOrder": map[string]any{"id": order.ID.String()},
		},
	})

	for _, old := range []*models.CanvasRun{run, child} {
		require.NoError(t, router.LockAndProcessEvent(logger, *replacement, time.Now()))
		stored, err := models.FindCanvasEvent(replacement.ID)
		require.NoError(t, err)
		require.Equal(t, models.CanvasEventStatePending, stored.State, "replacement analysis must wait for the cancelled run subtree")
		var queued int64
		require.NoError(t, db.Model(&models.CanvasNodeQueueItem{}).Where("event_id = ?", replacement.ID).Count(&queued).Error)
		require.Zero(t, queued, "spare capacity must not allow replacement work before cancellation completes")
		require.NoError(t, NewRunFinalizer("", r.Registry).finalizeRun(old.WorkflowID, old.ID, runFinalizerTriggerSweep))
	}

	require.NoError(t, router.LockAndProcessEvent(logger, *replacement, time.Now()))
	stored, err := models.FindCanvasEvent(replacement.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, stored.State)
	session, err := models.FindPlanningSessionByRun(db, replacement.RunID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, *session.DraftWorkOrderID)
}

func createBacklogAdmissionEvent(t *testing.T, r *support.ResourceRegistry) (*models.CanvasEvent, *models.FactoryWorkOrder) {
	t.Helper()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("backlog"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: models.FactoryAppBacklogTriggerID, Type: models.NodeTypeTrigger,
			Ref:      datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onWorkOrder"}}),
			Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 4)),
		},
		{NodeID: "refine-task", Type: models.NodeTypeComponent},
	}, []models.Edge{{SourceID: models.FactoryAppBacklogTriggerID, TargetID: "refine-task", Channel: "default"}})
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	order, err := factory.CreateWorkOrder(db, "Refine task", "", &r.User, nil, nil)
	require.NoError(t, err)
	return support.EmitCanvasEventForNodeWithData(t, canvas.ID, models.FactoryAppBacklogTriggerID, "default", nil, map[string]any{
		"type": "workOrder.created",
		"data": map[string]any{
			models.WorkOrderCreatedRefinementEnabledDataKey: true,
			"workOrder": map[string]any{"id": order.ID.String()},
		},
	}), order
}

func TestRootEventsDoNotWaitForRunnerCapacity(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	setRunnerPlanLimit(t, r, 1)
	router := NewEventRouter("")
	for range 3 {
		event, _ := createBacklogAdmissionEvent(t, r)
		require.NoError(t, router.LockAndProcessEvent(log.NewEntry(log.New()), *event, time.Now()))
		stored, err := models.FindCanvasEvent(event.ID)
		require.NoError(t, err)
		require.Equal(t, models.CanvasEventStateRouted, stored.State)
	}
	var count int64
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasNodeQueueItem{}).Count(&count).Error)
	require.Equal(t, int64(3), count, "runner limits must not block automation submission")
}
