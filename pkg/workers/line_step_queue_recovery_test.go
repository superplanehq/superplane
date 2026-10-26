package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

// A stale line waiter must not roll back admission for a healthy later step.
// These invalid references can remain after edits or interrupted cleanup.
func TestLineStepQueueDiscardsInvalidWaitersAndStartsHealthyWork(t *testing.T) {
	for _, invalid := range []string{"version", "entrypoint", "deleted workspace", "missing dispatch"} {
		t.Run(invalid, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			badLine := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})
			goodLine := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})
			_, badBlocker := badLine.dispatchLine(t, badLine.createOpenWorkOrder(t, r, "Block first step"))
			_, goodBlocker := goodLine.dispatchLine(t, goodLine.createOpenWorkOrder(t, r, "Block second step"))
			badOrder := badLine.createOpenWorkOrder(t, r, "Invalid waiter")
			bad, badWait := badLine.dispatchLine(t, badOrder)
			goodOrder := goodLine.createOpenWorkOrder(t, r, "Healthy later waiter")
			good, goodWait := goodLine.dispatchLine(t, goodOrder)
			require.NotNil(t, badWait.QueueItem)
			require.NotNil(t, goodWait.QueueItem)

			// Release step capacity without invoking the finalizer or its sweep.
			for _, blocker := range []*models.FactoryLineStepResult{badBlocker, goodBlocker} {
				finishRun(t, blocker.Run, models.CanvasRunResultPassed)
				require.NoError(t, blocker.Execution.MarkFinished(db, models.CanvasRunResultPassed))
			}

			want := models.CanvasRunResultFailed
			switch invalid {
			case "version":
				otherApp, err := models.FindCanvasInTransaction(db, r.Organization.ID, goodLine.line.Steps[0].AppID)
				require.NoError(t, err)
				require.NoError(t, db.Model(&models.Canvas{}).Where("id = ?", bad.Steps[0].AppID).
					Update("live_version_id", otherApp.LiveVersionID).Error)
			case "entrypoint":
				bad.Steps[0].Entrypoint = "missing"
				require.NoError(t, db.Model(bad).Update("steps", bad.Steps).Error)
			case "deleted workspace":
				want = models.CanvasRunResultCancelled
				require.NoError(t, badLine.factory.SoftDelete(db))
			case "missing dispatch":
				// Reproduce an orphan left by cleanup that bypassed foreign keys.
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					if err := tx.Exec("SET LOCAL session_replication_role = replica").Error; err != nil {
						return err
					}
					return tx.Delete(bad).Error
				}))
			}
			require.NotNil(t, queueItemForDispatch(t, bad.ID), "the sweep must own recovery")

			NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())

			assert.Nil(t, queueItemForDispatch(t, bad.ID))
			assert.Empty(t, executionsForOrder(t, badOrder.ID))
			if invalid != "missing dispatch" {
				recovered := reloadDispatch(t, bad.ID)
				assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, recovered.State)
				assert.Equal(t, want, recovered.Result)
			}
			assert.Nil(t, queueItemForDispatch(t, good.ID))
			executions := executionsForOrder(t, goodOrder.ID)
			require.Len(t, executions, 1)
			assert.Equal(t, good.ID, executions[0].LineDispatchID)
			assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, executions[0].Status)
		})
	}
}
