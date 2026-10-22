package workers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func Test__OrganizationQueue_ConcurrentDispatchesReserveSharedSlots(t *testing.T) {
	r := support.Setup(t)
	setTaskPlanLimit(t, r, 3)
	workspaces := []*stepQueueFixture{
		setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)}),
		setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)}),
	}
	orders := make([]*models.FactoryWorkOrder, 12)
	for i := range orders {
		orders[i] = workspaces[i%2].createOpenWorkOrder(t, r, "Concurrent")
	}
	var wg sync.WaitGroup
	errors := make(chan error, len(orders))
	for i, order := range orders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
				_, _, err := workspaces[i%2].line.Dispatch(tx, order)
				return err
			})
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	assertOrganizationTaskCounts(t, r.Organization.ID, 3, 9)

	// An independently locked organization remains dispatchable.
	other := *r
	var err error
	other.Organization, err = models.CreateOrganizationInTransaction(database.DB(t.Context()), support.RandomName("other"), "")
	require.NoError(t, err)
	setTaskPlanLimit(t, &other, 1)
	independent := setupStepQueueLine(t, &other, []*int{stepMaxParallelism(10)})
	order := independent.createOpenWorkOrder(t, &other, "Independent")
	_, result := independent.dispatchLine(t, order)
	require.NotNil(t, result.Run)
	assertOrganizationTaskCounts(t, other.Organization.ID, 1, 0)
}

func Test__OrganizationQueue_PlanChangesAndConcurrentSweeps(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	for range 4 {
		fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Task"))
	}
	assertOrganizationTaskCounts(t, r.Organization.ID, 1, 3)
	plan.ParallelTaskLimit = stepMaxParallelism(3)
	require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
	finalizer := NewRunFinalizer("", r.Registry)
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() { finalizer.sweepQueuedTasks(t.Context()) })
	}
	wg.Wait()
	assertOrganizationTaskCounts(t, r.Organization.ID, 3, 1)
	plan.ParallelTaskLimit = stepMaxParallelism(1)
	require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
	finalizer.sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 3, 1)
	setTaskPlanLimit(t, r, 4)
	finalizer.sweepQueuedTasks(t.Context())
	finalizer.sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 4, 0)
}

func Test__OrganizationQueue_SkipsLockedOrganization(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	for range 2 {
		fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Task"))
	}
	plan.ParallelTaskLimit = stepMaxParallelism(2)
	require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
	locked := database.DB(t.Context()).Begin()
	require.NoError(t, locked.Error)
	defer locked.Rollback()
	require.NoError(t, models.LockOrganizationTaskAdmission(locked, r.Organization.ID))
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(ctx)
	require.NoError(t, ctx.Err(), "sweep must skip the lock without waiting")
	assertOrganizationTaskCounts(t, r.Organization.ID, 1, 1)
	require.NoError(t, locked.Commit().Error)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 2, 0)
}

func Test__OrganizationQueue_OldestEligibleSkipsLocallyFullStep(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	blocked := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})
	eligible := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	blocked.dispatchLine(t, blocked.createOpenWorkOrder(t, r, "Running"))
	older, _ := blocked.dispatchLine(t, blocked.createOpenWorkOrder(t, r, "Locally blocked"))
	first, _ := eligible.dispatchLine(t, eligible.createOpenWorkOrder(t, r, "Oldest eligible"))
	second, _ := eligible.dispatchLine(t, eligible.createOpenWorkOrder(t, r, "Second eligible"))
	// Equal timestamps exercise the ID tie-break independently of wall clock.
	require.NoError(t, database.DB(t.Context()).Model(&models.FactoryWorkOrderQueueItem{}).
		Where("line_dispatch_id IN ?", []uuid.UUID{first.ID, second.ID}).Update("created_at", time.Now().Add(-time.Minute)).Error)
	require.NoError(t, database.DB(t.Context()).Model(&models.FactoryWorkOrderQueueItem{}).
		Where("line_dispatch_id = ?", older.ID).Update("created_at", time.Now().Add(-2*time.Minute)).Error)
	firstItem, secondItem := queueItemForDispatch(t, first.ID), queueItemForDispatch(t, second.ID)
	want, waits := first, second
	if firstItem.ID.String() > secondItem.ID.String() {
		want, waits = second, first
	}
	plan.ParallelTaskLimit = stepMaxParallelism(2)
	require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
	// A new dispatch must not take the free slot before existing eligible work.
	newcomer, result := eligible.dispatchLine(t, eligible.createOpenWorkOrder(t, r, "Newcomer"))
	require.NotNil(t, result.QueueItem)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assert.Nil(t, queueItemForDispatch(t, want.ID))
	assert.NotNil(t, queueItemForDispatch(t, waits.ID))
	assert.NotNil(t, queueItemForDispatch(t, older.ID))
	assert.NotNil(t, queueItemForDispatch(t, newcomer.ID))
	assertOrganizationTaskCounts(t, r.Organization.ID, 2, 3)
}

func Test__OrganizationQueue_InvalidCandidatesDoNotBlockValidWork(t *testing.T) {
	for _, invalid := range []string{"index", "app", "version", "entrypoint", "closed", "deleted", "finished", "missing"} {
		t.Run(invalid, func(t *testing.T) {
			r := support.Setup(t)
			plan := setTaskPlanLimit(t, r, 1)
			fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
			valid := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
			fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Running"))
			badOrder := fixture.createOpenWorkOrder(t, r, "Invalid")
			bad, _ := fixture.dispatchLine(t, badOrder)
			good, _ := valid.dispatchLine(t, valid.createOpenWorkOrder(t, r, "Valid"))
			db := database.DB(t.Context())
			wantResult := models.CanvasRunResultFailed
			switch invalid {
			case "index":
				require.NoError(t, db.Model(&models.FactoryWorkOrderQueueItem{}).Where("line_dispatch_id = ?", bad.ID).Update("step_index", -1).Error)
			case "app":
				require.NoError(t, db.Model(&models.Canvas{}).Where("id = ?", bad.Steps[0].AppID).Update("deleted_at", time.Now()).Error)
			case "version":
				otherApp, err := models.FindCanvasInTransaction(db, r.Organization.ID, valid.line.Steps[0].AppID)
				require.NoError(t, err)
				// The reference exists, but it is not a version of this app.
				require.NoError(t, db.Model(&models.Canvas{}).Where("id = ?", bad.Steps[0].AppID).Update("live_version_id", otherApp.LiveVersionID).Error)
			case "entrypoint":
				bad.Steps[0].Entrypoint = "missing"
				require.NoError(t, db.Model(bad).Update("steps", bad.Steps).Error)
			case "closed":
				wantResult = models.CanvasRunResultCancelled
				require.NoError(t, db.Model(badOrder).Update("state", models.FactoryWorkOrderStateClosed).Error)
			case "deleted":
				wantResult = models.CanvasRunResultCancelled
				require.NoError(t, fixture.factory.SoftDelete(db))
			case "finished":
				wantResult = models.CanvasRunResultPassed
				require.NoError(t, bad.Finish(db, wantResult))
			case "missing":
				// Reproduce stale storage from a cleanup that bypassed foreign keys.
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					if err := tx.Exec("SET LOCAL session_replication_role = replica").Error; err != nil {
						return err
					}
					return tx.Delete(bad).Error
				}))
			}
			plan.ParallelTaskLimit = stepMaxParallelism(2)
			require.NoError(t, models.SaveBillingPlan(db, plan))
			NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
			assert.Nil(t, queueItemForDispatch(t, bad.ID))
			assert.Nil(t, queueItemForDispatch(t, good.ID))
			assert.Empty(t, executionsForOrder(t, badOrder.ID))
			if invalid != "missing" {
				assert.Equal(t, wantResult, reloadDispatch(t, bad.ID).Result)
			}
			assertOrganizationTaskCounts(t, r.Organization.ID, 2, 0)
		})
	}
}

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

func Test__OrganizationQueue_UnlimitedPlansPreserveStepCapacity(t *testing.T) {
	for _, billing := range []string{"yes", "no"} {
		t.Run(billing, func(t *testing.T) {
			r := support.Setup(t)
			plan := setTaskPlanLimit(t, r, 1)
			if billing == "yes" {
				plan.ParallelTaskLimit = nil
				require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
			}
			t.Setenv(models.BillingEnabledEnv, billing)
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

func Test__OrganizationQueue_RetryAndReplacementAdmitOlderWorkspaceFirst(t *testing.T) {
	for _, scenario := range []struct{ operation, state string }{
		{"retry", models.CanvasRunStatePending}, {"replace", models.CanvasRunStatePending},
		{"retry", models.CanvasRunStateStarted}, {"replace", models.CanvasRunStateStarted},
	} {
		t.Run(scenario.operation+"/"+scenario.state, func(t *testing.T) {
			r := support.Setup(t)
			setTaskPlanLimit(t, r, 1)
			one := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10), stepMaxParallelism(10)})
			two := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
			order := one.createOpenWorkOrder(t, r, "Rerun")
			_, running := one.dispatchLine(t, order)
			finishRun(t, running.Run, models.CanvasRunResultPassed)
			next := advanceFactoryLine(t, r, running.Run.ID)
			require.Len(t, next, 1)
			oldRun, err := models.FindUnscopedCanvasRun(database.DB(t.Context()), next[0].runID)
			require.NoError(t, err)
			if scenario.state == models.CanvasRunStateStarted {
				require.NoError(t, NewRunInitializer("", r.Registry).initializeRun(oldRun.WorkflowID, oldRun.ID, runInitializerTriggerPending))
			}
			waiter, _ := two.dispatchLine(t, two.createOpenWorkOrder(t, r, "Older"))
			var started []*models.FactoryLineStepResult
			require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
				if err := models.LockOrganizationTaskAdmission(tx, r.Organization.ID); err != nil {
					return err
				}
				if err := order.LockForUpdate(tx); err != nil {
					return err
				}
				if scenario.operation == "retry" {
					_, results, err := order.RetryLineStep(tx, one.line, 1)
					started = results
					return err
				}
				var err error
				started, err = order.AbandonActiveLineDispatch(tx)
				if err != nil {
					return err
				}
				_, _, err = one.line.Dispatch(tx, order)
				return err
			}))
			oldRun, err = models.FindUnscopedCanvasRun(database.DB(t.Context()), oldRun.ID)
			for _, result := range started {
				assert.Nil(t, result.Run, "replacement must not start another run before cancellation finishes")
			}
			require.NoError(t, err)
			require.Equal(t, models.CanvasRunStateCancelling, oldRun.State)
			require.NoError(t, NewRunInitializer("", r.Registry).initializeRun(oldRun.WorkflowID, oldRun.ID, runInitializerTriggerPending))
			NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
			assert.NotNil(t, queueItemForDispatch(t, waiter.ID), "cancelling runs retain organization capacity")
			require.NoError(t, NewRunFinalizer("", r.Registry).finalizeRun(oldRun.WorkflowID, oldRun.ID, runFinalizerTriggerSweep))
			assert.Nil(t, queueItemForDispatch(t, waiter.ID))
			active, err := order.FindActiveLineDispatch(database.DB(t.Context()))
			require.NoError(t, err)
			assert.NotNil(t, queueItemForDispatch(t, active.ID))
			assertOrganizationTaskCounts(t, r.Organization.ID, 1, 1)
		})
	}
}

func Test__OrganizationQueue_FullCapacityDoesNotLoadCandidates(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Running"))
	bad, _ := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Unreadable waiter"))
	db := database.DB(t.Context())
	require.NoError(t, db.Exec("UPDATE factory_work_order_line_dispatches SET steps = '[42]'::jsonb WHERE id = ?", bad.ID).Error)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := models.AdmitQueuedForOrganization(tx, r.Organization.ID)
		return err
	}))
	assert.NotNil(t, queueItemForDispatch(t, bad.ID))
	plan.ParallelTaskLimit = stepMaxParallelism(2)
	require.NoError(t, models.SaveBillingPlan(db, plan))
	require.Error(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := models.AdmitQueuedForOrganization(tx, r.Organization.ID)
		return err
	}), "the unreadable candidate is only inspected when capacity is available")
}

func Test__OrganizationQueue_SweepBoundsAdmissionsAndContinuesNextPass(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(200)})
	for range 102 {
		fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Backlog"))
	}
	plan.ParallelTaskLimit = stepMaxParallelism(200)
	require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
	finalizer := NewRunFinalizer("", r.Registry)
	finalizer.sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 101, 1)
	finalizer.sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 102, 0)
}

func Test__OrganizationQueue_InitializationFailureReleasesCapacityAcrossWorkspaces(t *testing.T) {
	r := support.Setup(t)
	setTaskPlanLimit(t, r, 1)
	one := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	two := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	_, running := one.dispatchLine(t, one.createOpenWorkOrder(t, r, "Fails initialization"))
	waiter, _ := two.dispatchLine(t, two.createOpenWorkOrder(t, r, "Waiting"))
	running.Run.Callbacks[0].On = "bogus"
	require.NoError(t, database.DB(t.Context()).Model(running.Run).Update("callbacks", running.Run.Callbacks).Error)
	require.NoError(t, NewRunInitializer("", r.Registry).initializeRun(running.Run.WorkflowID, running.Run.ID, runInitializerTriggerPending))
	assert.Nil(t, queueItemForDispatch(t, waiter.ID))
	assertOrganizationTaskCounts(t, r.Organization.ID, 1, 0)
}

func Test__OrganizationQueue_UnclassifiedErrorRollsBackOrganization(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Running"))
	good, _ := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "First valid"))
	bad, _ := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Unreadable snapshot"))
	db := database.DB(t.Context())
	require.NoError(t, db.Exec("UPDATE factory_work_order_line_dispatches SET steps = '[42]'::jsonb WHERE id = ?", bad.ID).Error)
	plan.ParallelTaskLimit = stepMaxParallelism(3)
	require.NoError(t, models.SaveBillingPlan(db, plan))
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := models.AdmitQueuedForOrganization(tx, r.Organization.ID)
		return err
	})
	require.Error(t, err)
	assertOrganizationTaskCounts(t, r.Organization.ID, 1, 2)
	assert.NotNil(t, queueItemForDispatch(t, good.ID))
	var runs int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ?", fixture.line.Steps[0].AppID).Count(&runs).Error)
	assert.Equal(t, int64(1), runs, "even a previously admitted candidate must roll back without orphan runs")
	require.NoError(t, db.Model(bad).Update("steps", bad.Steps).Error)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 3, 0)
}

func Test__OrganizationQueue_CloseAndReplacementRaceWithSweep(t *testing.T) {
	for _, operation := range []string{"close", "replace"} {
		t.Run(operation, func(t *testing.T) {
			r := support.Setup(t)
			plan := setTaskPlanLimit(t, r, 1)
			fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
			fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Running"))
			order := fixture.createOpenWorkOrder(t, r, "Racing")
			fixture.dispatchLine(t, order)
			plan.ParallelTaskLimit = stepMaxParallelism(2)
			require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			var wg sync.WaitGroup
			failures := make(chan error, 1)
			wg.Go(func() { NewRunFinalizer("", r.Registry).sweepQueuedTasks(ctx) })
			wg.Go(func() {
				if operation == "close" {
					_, err := order.Close(database.DB(ctx), models.FactoryWorkOrderResultRejected, &r.User)
					failures <- err
					return
				}
				_, _, err := factoryactions.DispatchWorkOrderOnLine(database.DB(ctx), fixture.factory, order.ID, fixture.line, &r.User, 0, true, "", "")
				failures <- err
			})
			wg.Wait()
			require.NoError(t, <-failures)
			require.NoError(t, ctx.Err())
			before := len(executionsForOrder(t, order.ID))
			NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
			if operation == "close" {
				assert.Len(t, executionsForOrder(t, order.ID), before, "closed work must not restart")
			}
			var active int64
			require.NoError(t, database.DB(t.Context()).Model(&models.FactoryWorkOrderExecution{}).
				Where("organization_id = ? AND status IN ?", r.Organization.ID, []string{models.FactoryWorkOrderExecutionStatusPending, models.FactoryWorkOrderExecutionStatusRunning}).Count(&active).Error)
			assert.LessOrEqual(t, active, int64(2))
		})
	}
}

func Test__OrganizationQueue_SweepPublishesAdmittedWorkspaceAfterCommit(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	one := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	two := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	one.dispatchLine(t, one.createOpenWorkOrder(t, r, "Running"))
	order := two.createOpenWorkOrder(t, r, "Waiting")
	two.dispatchLine(t, order)
	url, err := config.RabbitMQURL()
	require.NoError(t, err)
	connection, err := amqp.Dial(url)
	require.NoError(t, err)
	defer connection.Close()
	channel, err := connection.Channel()
	require.NoError(t, err)
	defer channel.Close()
	queue, err := channel.QueueDeclare("", false, true, true, false, nil)
	require.NoError(t, err)
	require.NoError(t, channel.QueueBind(queue.Name, messages.FactoryWorkOrderUpdatedRoutingKey, messages.CanvasExchange, false, nil))
	deliveries, err := channel.Consume(queue.Name, "", true, true, false, false, nil)
	require.NoError(t, err)
	plan.ParallelTaskLimit = stepMaxParallelism(2)
	require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), plan))
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case delivery := <-deliveries:
			var update pb.FactoryWorkOrderUpdatedMessage
			require.NoError(t, proto.Unmarshal(delivery.Body, &update))
			if update.OrderId != order.ID.String() {
				continue
			}
			assert.Equal(t, two.factory.ID.String(), update.FactoryId)
			require.Len(t, executionsForOrder(t, order.ID), 1, "the published execution must be committed")
			return
		case <-timer.C:
			t.Fatal("admitted workspace did not receive an update")
		}
	}
}

func Test__OrganizationQueue_InvalidWaiterPreservesExecutionOwningDispatch(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10), stepMaxParallelism(10)})
	order := fixture.createOpenWorkOrder(t, r, "Still owned")
	dispatch, running := fixture.dispatchLine(t, order)
	db := database.DB(t.Context())
	// The running step already holds this task's slot, so the next step would
	// start. Store the invalid waiter directly to exercise sweep cleanup.
	require.NoError(t, db.Create(&models.FactoryWorkOrderQueueItem{
		ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: fixture.factory.ID,
		WorkOrderID: order.ID, LineID: dispatch.LineID, LineDispatchID: dispatch.ID,
		StepIndex: 999, StepName: "invalid", CreatedAt: time.Now(),
	}).Error)
	plan.ParallelTaskLimit = stepMaxParallelism(2)
	require.NoError(t, models.SaveBillingPlan(db, plan))
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assert.Nil(t, queueItemForDispatch(t, dispatch.ID))
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, reloadDispatch(t, dispatch.ID).State)
	current, err := models.FindWorkOrderExecutionByRunID(db, running.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, current.Status)
}

func Test__OrganizationQueue_LineAdmissionDoesNotPassOlderRootEvent(t *testing.T) {
	r := support.Setup(t)
	plan := setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Holder"))
	_, older := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Older line"))
	require.NotNil(t, older.QueueItem)
	root, _ := createBacklogAdmissionEvent(t, r)
	_, newer := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Newer line"))
	require.NotNil(t, newer.QueueItem)

	plan.ParallelTaskLimit = stepMaxParallelism(3)
	db := database.DB(t.Context())
	require.NoError(t, models.SaveBillingPlan(db, plan))
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())

	assert.Nil(t, queueItemForDispatch(t, older.QueueItem.LineDispatchID), "the oldest line must start")
	assert.NotNil(t, queueItemForDispatch(t, newer.QueueItem.LineDispatchID), "a newer line must not pass an older root event")
	var stored models.CanvasEvent
	require.NoError(t, db.First(&stored, "id = ?", root.ID).Error)
	assert.Equal(t, models.CanvasEventStatePending, stored.State)
}

func Test__OrganizationQueue_ActiveTaskStartsBehindInactiveBatch(t *testing.T) {
	r := support.Setup(t)
	setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10), stepMaxParallelism(10)})
	order := fixture.createOpenWorkOrder(t, r, "Active")
	dispatch, running := fixture.dispatchLine(t, order)
	require.NotNil(t, running.Run)
	for range 100 {
		_, queued := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Waiting"))
		require.NotNil(t, queued.QueueItem)
	}
	db := database.DB(t.Context())
	require.NoError(t, db.Create(&models.FactoryWorkOrderQueueItem{
		ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: fixture.factory.ID,
		WorkOrderID: order.ID, LineID: dispatch.LineID, LineDispatchID: dispatch.ID,
		StepIndex: 1, StepName: "next", CreatedAt: time.Now(),
	}).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := models.AdmitQueuedForOrganization(tx, r.Organization.ID)
		return err
	}))

	assert.Nil(t, queueItemForDispatch(t, dispatch.ID), "the active task must start the step it already holds a slot for")
	assert.Len(t, executionsForOrder(t, order.ID), 2)
	assertOrganizationTaskCounts(t, r.Organization.ID, 2, 100)
}
