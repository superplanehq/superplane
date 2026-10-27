package workers

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

func TestSuspensionDiscardsScheduleEventsAndRetainsFutureTicks(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "schedule", Type: models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
		Configuration: datatypes.NewJSONType(map[string]any{"type": "hours", "hoursInterval": 1, "minute": 0}),
	}}, nil)
	request := models.CanvasNodeRequest{ID: uuid.New(), WorkflowID: canvas.ID, NodeID: "schedule", Type: models.NodeRequestTypeInvokeAction, State: models.NodeExecutionRequestStatePending,
		RunAt: time.Now().Add(-time.Minute), Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{ActionName: "emitEvent"}})}
	require.NoError(t, db.Create(&request).Error)
	require.NoError(t, r.Organization.Suspend(db, r.Account.ID, "Unattended automation"))
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
	require.NoError(t, executor.Execute(request))
	var events int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&events).Error)
	require.Zero(t, events)
	var next models.CanvasNodeRequest
	require.NoError(t, db.Where("workflow_id = ? AND state = ?", canvas.ID, models.NodeExecutionRequestStatePending).First(&next).Error)
	require.True(t, next.RunAt.After(time.Now()))
	worker := NewOrganizationSuspensionWorker(nil)
	worker.processPending(t.Context())
	require.NoError(t, r.Organization.Reactivate(db, r.Account.ID))
	// The retained schedule produces only a future run after reactivation.
	require.NoError(t, executor.RunNow(next))
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&events).Error)
	require.EqualValues(t, 1, events)
	history, err := models.ListOrganizationSuspensionEvents(db, r.Organization.ID)
	require.NoError(t, err)
	for _, audit := range history {
		if audit.Action == "suspend" {
			require.NoError(t, worker.process(t.Context(), audit))
		}
	}
	var activeRuns int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ? AND state = ?", canvas.ID, models.CanvasRunStateStarted).Count(&activeRuns).Error)
	require.EqualValues(t, 1, activeRuns, "a stale worker must not cancel work created after reactivation")

}

func TestSuspensionCancelsExistingWorkAfterWorkerRestart(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "start", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})}}, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	require.NoError(t, r.Organization.Suspend(db, r.Account.ID, "Abuse"))
	worker := NewOrganizationSuspensionWorker(nil)
	worker.processPending(t.Context())
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, event.RunID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasRunStateCancelling, run.State)
	finalizer := NewRunFinalizer("", r.Registry)
	_, _, err = finalizer.maybeFinalizeRun(db, run.ID, runFinalizerTriggerSweep, nil, nil)
	require.NoError(t, err)
	// A new worker reads the pending cancellation from the database.
	NewOrganizationSuspensionWorker(nil).processPending(t.Context())
	require.NoError(t, r.Organization.Reactivate(db, r.Account.ID))
}
