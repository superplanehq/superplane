package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunQueueWaitPersistsOverlappingNestedAndRequeuedIntervals(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "work", Type: models.NodeTypeComponent}}, nil)
	child := createRunRecord(t, canvas.ID, "work", nil)
	grandchild := createRunRecord(t, canvas.ID, "work", &child.ID)
	unrelated := createRunRecord(t, canvas.ID, "work", nil)
	fleet := models.DefaultInstallationRunnerFleets("dev")[0]
	require.NoError(t, fleet.Create(db))
	base := time.Now().Add(-10 * time.Minute).Truncate(time.Microsecond)
	queue := func(run *models.CanvasRun, seconds int) *models.RunnerTask {
		execution := createRunExecution(t, canvas.ID, run.ID, "work")
		at := base.Add(time.Duration(seconds) * time.Second)
		task := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
			Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
			PayloadCiphertext: []byte("encrypted"), QueuedAt: at, CreatedAt: at, UpdatedAt: at}
		require.NoError(t, db.Create(task).Error)
		require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, "work", execution.ID, "task_id", task.ID.String()))
		return task
	}
	first, second := queue(child, 0), queue(child, 20)
	require.NoError(t, first.RequestCancel(db, base.Add(40*time.Second)))
	require.NoError(t, second.RequestCancel(db, base.Add(60*time.Second)))
	nested := queue(grandchild, 80)
	require.NoError(t, nested.RequestCancel(db, base.Add(100*time.Second)))
	require.NoError(t, db.Model(nested).Updates(map[string]any{"state": models.RunnerTaskStateQueued, "finished_at": nil, "updated_at": base.Add(120 * time.Second)}).Error)
	nested, err := models.FindRunnerTask(db, nested.ID)
	require.NoError(t, err)
	require.NoError(t, nested.RequestCancel(db, base.Add(130*time.Second)))
	queue(unrelated, -10)
	duration, open, err := child.RunnerQueueWait(db, base, base.Add(150*time.Second))
	require.NoError(t, err)
	require.Equal(t, 90*time.Second, duration)
	require.False(t, open)
	last := queue(grandchild, 140)
	duration, open, err = child.RunnerQueueWait(db, base, base.Add(150*time.Second))
	require.NoError(t, err)
	require.Equal(t, 100*time.Second, duration)
	require.True(t, open)
	// Reservation closes the wait; provisioning and preparation stay charged.
	require.NoError(t, db.Model(last).Updates(map[string]any{"state": models.RunnerTaskStateReserved, "updated_at": base.Add(145 * time.Second)}).Error)
	duration, open, err = child.RunnerQueueWait(db, base, base.Add(150*time.Second))
	require.NoError(t, err)
	require.Equal(t, 95*time.Second, duration)
	require.False(t, open)
}
