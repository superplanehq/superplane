package workers

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
	"testing"
)

func assertOrganizationTaskCounts(t *testing.T, orgID uuid.UUID, active, queued int64) {
	t.Helper()
	var executions, items int64
	db := database.DB(t.Context())
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).Where("organization_id = ? AND status IN ?", orgID,
		[]string{models.FactoryWorkOrderExecutionStatusPending, models.FactoryWorkOrderExecutionStatusRunning}).Count(&executions).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderQueueItem{}).Where("organization_id = ?", orgID).Count(&items).Error)
	assert.Equal(t, active, executions)
	assert.Equal(t, queued, items)
}

func TestRunnerLimitsPreserveStepCapacity(t *testing.T) {
	for _, billing := range []string{"limited", "unlimited", "disabled"} {
		t.Run(billing, func(t *testing.T) {
			r := support.Setup(t)
			plan := setRunnerPlanLimit(t, r, 1)
			if billing == "unlimited" {
				plan.ParallelTaskLimit = nil
				require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
			}
			t.Setenv(models.BillingEnabledEnv, "yes")
			if billing == "disabled" {
				t.Setenv(models.BillingEnabledEnv, "no")
			}
			one := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})
			two := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})
			one.dispatchLine(t, one.createOpenWorkOrder(t, r, "First"))
			_, queued := one.dispatchLine(t, one.createOpenWorkOrder(t, r, "Locally blocked"))
			require.NotNil(t, queued.QueueItem)
			_, started := two.dispatchLine(t, two.createOpenWorkOrder(t, r, "Other workspace"))
			require.NotNil(t, started.Run)
			assertOrganizationTaskCounts(t, r.Organization.ID, 2, 1)
		})
	}
}

func TestOrganizationQueueReplacementWaitsForCancellationWithSpareCapacity(t *testing.T) {
	for _, operation := range []string{"retry", "replace"} {
		t.Run(operation, func(t *testing.T) {
			r := support.Setup(t)
			setRunnerPlanLimit(t, r, 2)
			fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
			order := fixture.createOpenWorkOrder(t, r, "Replace after cancellation")
			_, running := fixture.dispatchLine(t, order)
			db := database.DB(t.Context())
			child, err := models.CreateCanvasRunInTransaction(db, running.Run.WorkflowID, running.Run.NodeID, models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			require.NoError(t, db.Model(child).Update("parent_run_id", running.Run.ID).Error)

			var replacement *models.FactoryLineStepResult
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				if operation == "retry" {
					var err error
					replacement, _, err = order.RetryLineStep(tx, fixture.line, 0)
					return err
				}
				if _, err := order.AbandonActiveLineDispatch(tx); err != nil {
					return err
				}
				var err error
				_, replacement, err = fixture.line.Dispatch(tx, order)
				return err
			}))
			require.NotNil(t, replacement.QueueItem, "spare capacity must not start replacement work while its old run is stopping")
			finalizer := NewRunFinalizer("", r.Registry)
			finalizer.sweepQueuedTasks(t.Context())
			require.Len(t, executionsForOrder(t, order.ID), 1, "the sweep must keep the replacement queued")
			_, unrelated := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Use the spare slot"))
			require.NotNil(t, unrelated.Run, "a stopping task must not block unrelated eligible work")

			require.NoError(t, finalizer.finalizeRun(running.Run.WorkflowID, running.Run.ID, runFinalizerTriggerSweep))
			stored, err := models.FindUnscopedCanvasRun(db, running.Run.ID)
			require.NoError(t, err)
			require.Equal(t, models.CanvasRunStateFinished, stored.State)
			finalizer.sweepQueuedTasks(t.Context())
			require.Len(t, executionsForOrder(t, order.ID), 1, "the replacement must wait for the cancelled descendant after its root finishes")
			require.NotNil(t, queueItemForDispatch(t, replacement.QueueItem.LineDispatchID))

			require.NoError(t, finalizer.finalizeRun(child.WorkflowID, child.ID, runFinalizerTriggerSweep))
			finalizer.sweepQueuedTasks(t.Context())
			assert.Nil(t, queueItemForDispatch(t, replacement.QueueItem.LineDispatchID))
			assert.Len(t, executionsForOrder(t, order.ID), 2, "the replacement starts after the old run subtree stops")
		})
	}
}

func Test__OrganizationQueue_InvalidWaiterPreservesExecutionOwningDispatch(t *testing.T) {
	r := support.Setup(t)
	setRunnerPlanLimit(t, r, 2)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10), stepMaxParallelism(1)})
	db := database.DB(t.Context())
	blocker := fixture.createOpenWorkOrder(t, r, "Step blocker")
	var blocking *models.FactoryLineStepResult
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, err := fixture.line.DispatchFrom(tx, blocker, 1)
		blocking = result
		return err
	}))
	order := fixture.createOpenWorkOrder(t, r, "Still owned")
	dispatch, running := fixture.dispatchLine(t, order)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		queued, err := dispatch.EnqueueOrStartStep(tx, order, 1)
		if err != nil {
			return err
		}
		require.NotNil(t, queued.QueueItem)
		return tx.Model(queued.QueueItem).Update("step_index", 999).Error
	}))
	finishRun(t, blocking.Run, models.CanvasRunResultPassed)
	require.NoError(t, blocking.Execution.MarkFinished(db, models.CanvasRunResultPassed))
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assert.Nil(t, queueItemForDispatch(t, dispatch.ID))
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, reloadDispatch(t, dispatch.ID).State)
	current, err := models.FindWorkOrderExecutionByRunID(db, running.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, current.Status)
}
