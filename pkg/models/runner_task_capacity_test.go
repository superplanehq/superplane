package models_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestRunnerCapacityConcurrentReservationsAcrossFleets(t *testing.T) {
	f := newRunnerCapacityTest(t, new(2))
	firstFleet := f.fleet("first-amd64")
	secondFleet := f.fleet("second-amd64")

	type reservation struct {
		runner    *models.Runner
		task      *models.RunnerTask
		automatic bool
	}
	var reservations []reservation
	for i := range 12 {
		fleet := firstFleet
		if i%2 != 0 {
			fleet = secondFleet
		}
		reservations = append(reservations, reservation{
			runner:    f.runner(fleet.ID),
			task:      f.task(f.organizationID, fleet.ID),
			automatic: i%2 == 0,
		})
	}

	errs := make([]error, len(reservations))
	start := make(chan struct{})
	var group sync.WaitGroup
	for i, candidate := range reservations {
		group.Go(func() {
			<-start
			errs[i] = f.db.Transaction(func(tx *gorm.DB) error {
				if candidate.automatic {
					_, err := candidate.runner.ReserveNextTask(tx)
					return err
				}
				return candidate.task.Reserve(tx, candidate.runner.ID)
			})
		})
	}
	close(start)
	group.Wait()

	for _, err := range errs {
		if errors.Is(err, models.ErrOrganizationRunnerCapacity) {
			continue
		}
		require.NoError(t, err)
	}
	var active int64
	require.NoError(t, f.db.Model(&models.RunnerTask{}).
		Where("organization_id = ? AND state IN ?", f.organizationID,
			[]string{models.RunnerTaskStateReserved, models.RunnerTaskStateRunning}).
		Count(&active).Error)
	assert.Positive(t, active)
	assert.LessOrEqual(t, active, int64(2))
}

func TestRunnerCapacityCountsJobsFromOneAutomationAndSeparatesOrganizations(t *testing.T) {
	f := newRunnerCapacityTest(t, new(2))
	fleet := f.fleet("shared-amd64")
	canvas, _ := support.CreateCanvas(t, f.organizationID, f.resource.User, nil, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "runner", "default", nil)

	for range 3 {
		task := f.task(f.organizationID, fleet.ID)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, "runner", event.ID, event.ID)
		require.NoError(t, models.CreateNodeExecutionKVInTransaction(f.db, canvas.ID, "runner", execution.ID, "task_id", task.ID.String()))
	}

	first, err := f.runner(fleet.ID).ReserveNextTask(f.db)
	require.NoError(t, err)
	second, err := f.runner(fleet.ID).ReserveNextTask(f.db)
	require.NoError(t, err)
	assert.NotEqual(t, first.Task.ID, second.Task.ID)
	empty, err := f.runner(fleet.ID).ReserveNextTask(f.db)
	require.NoError(t, err)
	assert.Nil(t, empty.Task)

	otherOrganization := support.CreateOrganization(t, f.resource, f.resource.User)
	_, err = billingmodels.SetAdminOrganizationPlan(f.db, otherOrganization.ID, f.plan.Key, nil)
	require.NoError(t, err)
	otherTask := f.task(otherOrganization.ID, fleet.ID)
	reserved, err := f.runner(fleet.ID).ReserveNextTask(f.db)
	require.NoError(t, err)
	assert.Equal(t, otherTask.ID, reserved.Task.ID)
}

func TestRunnerCapacitySkipsFullAndLockedOrganizationsWithoutRetainingLocks(t *testing.T) {
	for _, locked := range []bool{false, true} {
		name := "full organization"
		if locked {
			name = "locked organization"
		}
		t.Run(name, func(t *testing.T) {
			f := newRunnerCapacityTest(t, new(1))
			fleet := f.fleet("shared-amd64")
			otherOrganization := support.CreateOrganization(t, f.resource, f.resource.User)
			_, err := billingmodels.SetAdminOrganizationPlan(f.db, otherOrganization.ID, f.plan.Key, nil)
			require.NoError(t, err)

			if !locked {
				occupied := f.task(f.organizationID, fleet.ID)
				require.NoError(t, occupied.Reserve(f.db, f.runner(fleet.ID).ID))
			}
			firstSkipped := f.task(f.organizationID, fleet.ID)
			secondSkipped := f.task(f.organizationID, fleet.ID)
			firstEligible := f.task(otherOrganization.ID, fleet.ID)
			secondEligible := f.task(otherOrganization.ID, fleet.ID)
			// FIFO follows queue time even when task creation times differ.
			require.NoError(t, f.db.Model(firstEligible).Update("created_at", time.Now().Add(time.Hour)).Error)
			require.NoError(t, f.db.Model(secondEligible).Update("created_at", time.Now().Add(-time.Hour)).Error)
			runner := f.runner(fleet.ID)

			if locked {
				holder := f.db.Begin()
				require.NoError(t, holder.Error)
				defer holder.Rollback()
				_, err = models.LockOrganization(holder, f.organizationID)
				require.NoError(t, err)
			}

			tx := f.db.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			reserved, err := runner.ReserveNextTask(tx)
			require.NoError(t, err)
			assert.Equal(t, firstEligible.ID, reserved.Task.ID)
			// Keep the successful reservation open. Skipped candidates must still
			// be available to another transaction, rather than locked until commit.
			require.NoError(t, f.db.Transaction(func(probe *gorm.DB) error {
				for _, skipped := range []*models.RunnerTask{firstSkipped, secondSkipped} {
					if err := probe.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
						First(&models.RunnerTask{}, "id = ?", skipped.ID).Error; err != nil {
						return err
					}
				}
				if locked {
					return nil
				}
				return probe.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
					First(&models.Organization{}, "id = ?", f.organizationID).Error
			}))
		})
	}
}

func TestRunnerCapacitySerializesExplicitAdmissionBeforeRunnerLock(t *testing.T) {
	f := newRunnerCapacityTest(t, new(1))
	firstFleet := f.fleet("first-amd64")
	secondFleet := f.fleet("second-amd64")
	firstTask := f.task(f.organizationID, firstFleet.ID)
	firstRunner := f.runner(firstFleet.ID)
	secondTask := f.task(f.organizationID, secondFleet.ID)
	secondRunner := f.runner(secondFleet.ID)
	idleRunner := f.runner(secondFleet.ID)

	first := f.db.Begin()
	require.NoError(t, first.Error)
	defer first.Rollback()
	_, err := firstRunner.ReserveNextTask(first)
	require.NoError(t, err)
	storedFirst, err := models.FindRunnerTask(first, firstTask.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateReserved, storedFirst.State)

	empty, err := idleRunner.ReserveNextTask(f.db)
	require.NoError(t, err)
	assert.Nil(t, empty.Task)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	pid := make(chan int, 1)
	finished := make(chan struct{})
	var explicitError error
	go func() {
		defer close(finished)
		explicitError = f.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
			var backendPID int
			if err := conn.Raw("SELECT pg_backend_pid()").Scan(&backendPID).Error; err != nil {
				return err
			}
			pid <- backendPID
			return secondTask.Reserve(conn, secondRunner.ID)
		})
	}()
	defer func() {
		first.Rollback()
		cancel()
		<-finished
	}()
	var backendPID int
	select {
	case backendPID = <-pid:
	case <-finished:
		t.Fatalf("explicit reservation could not start: %v", explicitError)
	}
	require.Eventually(t, func() bool {
		var blocked bool
		return f.db.Raw("SELECT cardinality(pg_blocking_pids(?)) > 0", backendPID).
			Scan(&blocked).Error == nil && blocked
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, f.db.Transaction(func(probe *gorm.DB) error {
		return probe.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
			First(&models.Runner{}, "id = ?", secondRunner.ID).Error
	}))
	require.NoError(t, first.Commit().Error)
	<-finished
	require.ErrorIs(t, explicitError, models.ErrOrganizationRunnerCapacity)
	storedSecond, err := models.FindRunnerTask(f.db, secondTask.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, storedSecond.State)
	assert.Nil(t, storedSecond.RunnerID)
}

func TestRunnerCapacityReservationRetriesKeepTheSameSlot(t *testing.T) {
	f := newRunnerCapacityTest(t, new(1))
	fleet := f.fleet("retry-amd64")
	runner := f.runner(fleet.ID)
	task := f.task(f.organizationID, fleet.ID)
	staleTask := *task
	require.NoError(t, task.Reserve(f.db, runner.ID))
	require.NotNil(t, task.ReservedAt)
	reservedAt := *task.ReservedAt
	queued := f.task(f.organizationID, fleet.ID)

	for _, running := range []bool{false, true} {
		if running {
			require.NoError(t, task.Start(f.db, runner, "test-store", time.Now()))
		}
		require.NoError(t, staleTask.Reserve(f.db, runner.ID))
		require.NotNil(t, staleTask.ReservedAt)
		assert.WithinDuration(t, reservedAt, *staleTask.ReservedAt, time.Microsecond)
		retry, err := runner.ReserveNextTask(f.db)
		require.NoError(t, err)
		assert.Equal(t, task.ID, retry.Task.ID)
		assert.Equal(t, task.State, retry.Task.State)
		require.NotNil(t, retry.Task.ReservedAt)
		assert.WithinDuration(t, reservedAt, *retry.Task.ReservedAt, time.Microsecond)
		require.ErrorIs(t, queued.Reserve(f.db, f.runner(fleet.ID).ID), models.ErrOrganizationRunnerCapacity)
	}
}

func TestRunnerCapacityFollowsExistingLifecycleRelease(t *testing.T) {
	for _, release := range []string{"completion", "cancellation", "reserved runner loss", "running runner loss"} {
		t.Run(release, func(t *testing.T) {
			f := newRunnerCapacityTest(t, new(1))
			fleet := f.fleet("lifecycle-amd64")
			runner := f.runner(fleet.ID)
			task := f.task(f.organizationID, fleet.ID)
			queued := f.task(f.organizationID, fleet.ID)
			nextRunner := f.runner(fleet.ID)
			require.NoError(t, task.Reserve(f.db, runner.ID))

			if release == "cancellation" {
				require.NoError(t, task.RequestCancel(f.db, time.Now()))
				assert.Equal(t, models.RunnerTaskStateReserved, task.State)
				require.ErrorIs(t, queued.Reserve(f.db, nextRunner.ID), models.ErrOrganizationRunnerCapacity)
			}
			if release != "reserved runner loss" {
				require.NoError(t, task.Start(f.db, runner, "test-store", time.Now()))
			}
			if release == "cancellation" {
				require.NoError(t, task.RequestCancel(f.db, time.Now()))
				assert.Equal(t, models.RunnerTaskStateRunning, task.State)
			}
			require.ErrorIs(t, queued.Reserve(f.db, nextRunner.ID), models.ErrOrganizationRunnerCapacity)

			if release == "completion" || release == "cancellation" {
				completed, err := task.Complete(f.db, runner, "completion", datatypes.JSON([]byte(`{}`)), 0, "", release == "cancellation", time.Now())
				require.NoError(t, err)
				require.True(t, completed)
			} else {
				now := time.Now()
				lastSeenAt := now.Add(-2 * time.Minute)
				require.NoError(t, f.db.Model(runner).Update("last_seen_at", lastSeenAt).Error)
				require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
					locked, err := models.LockRunner(tx, runner.ID)
					if err != nil {
						return err
					}
					_, err = locked.MarkLost(tx, now.Add(-time.Minute), now)
					return err
				}))
			}
			require.NoError(t, queued.Reserve(f.db, nextRunner.ID))
			assert.Equal(t, models.RunnerTaskStateReserved, queued.State)
		})
	}
}

func TestRunnerCapacityRegistrationExpiryReleasesReservations(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		name := "uncanceled reservation returns to queue"
		if canceled {
			name = "canceled reservation finishes"
		}
		t.Run(name, func(t *testing.T) {
			f := newRunnerCapacityTest(t, new(1))
			fleet := f.fleet("expiry-amd64")
			runner := f.runner(fleet.ID)
			require.NoError(t, f.db.Model(runner).Update("state", models.RunnerStatePending).Error)
			task := f.task(f.organizationID, fleet.ID)
			queued := f.task(f.organizationID, fleet.ID)
			nextRunner := f.runner(fleet.ID)
			require.NoError(t, task.Reserve(f.db, runner.ID))
			if canceled {
				require.NoError(t, task.RequestCancel(f.db, time.Now()))
			}
			require.ErrorIs(t, queued.Reserve(f.db, nextRunner.ID), models.ErrOrganizationRunnerCapacity)

			now := time.Now()
			require.NoError(t, f.db.Create(&models.RunnerRegistration{
				JTI:       uuid.New(),
				RunnerID:  runner.ID,
				ExpiresAt: now.Add(-time.Minute),
				CreatedAt: now.Add(-2 * time.Minute),
			}).Error)
			require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
				locked, err := models.LockRunner(tx, runner.ID)
				if err != nil {
					return err
				}
				return locked.Expire(tx, now)
			}))

			stored, err := models.FindRunnerTask(f.db, task.ID)
			require.NoError(t, err)
			assert.Nil(t, stored.RunnerID)
			assert.Nil(t, stored.ReservedAt)
			expectedNext := task.ID
			if canceled {
				assert.Equal(t, models.RunnerTaskStateCanceled, stored.State)
				assert.True(t, stored.IsTerminal())
				require.NotNil(t, stored.FinishedAt)
				require.NotNil(t, stored.CancelRequestedAt)
				assert.False(t, stored.FinishedAt.Before(*stored.CancelRequestedAt))
				require.ErrorIs(t, stored.Reserve(f.db, nextRunner.ID), models.ErrRunnerTaskNotReservable)
				expectedNext = queued.ID
			} else {
				assert.Equal(t, models.RunnerTaskStateQueued, stored.State)
				assert.Nil(t, stored.FinishedAt)
			}
			reserved, err := nextRunner.ReserveNextTask(f.db)
			require.NoError(t, err)
			assert.Equal(t, expectedNext, reserved.Task.ID)
		})
	}
}

func TestRunnerCapacityReadsPlanChangesWithoutRevokingReservations(t *testing.T) {
	f := newRunnerCapacityTest(t, new(2))
	fleet := f.fleet("plan-amd64")
	first := f.task(f.organizationID, fleet.ID)
	second := f.task(f.organizationID, fleet.ID)
	firstRunner := f.runner(fleet.ID)
	secondRunner := f.runner(fleet.ID)
	require.NoError(t, first.Reserve(f.db, firstRunner.ID))
	require.NoError(t, first.Start(f.db, firstRunner, "test-store", time.Now()))
	require.NoError(t, second.Reserve(f.db, secondRunner.ID))
	require.NoError(t, second.Start(f.db, secondRunner, "test-store", time.Now()))
	queued := f.task(f.organizationID, fleet.ID)
	nextRunner := f.runner(fleet.ID)

	f.plan.ParallelTaskLimit = new(1)
	require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
	require.NoError(t, first.Reserve(f.db, firstRunner.ID))
	require.NoError(t, second.Reserve(f.db, secondRunner.ID))
	require.ErrorIs(t, queued.Reserve(f.db, nextRunner.ID), models.ErrOrganizationRunnerCapacity)
	completed, err := first.Complete(f.db, firstRunner, "first", datatypes.JSON([]byte(`{}`)), 0, "", false, time.Now())
	require.NoError(t, err)
	require.True(t, completed)
	require.ErrorIs(t, queued.Reserve(f.db, nextRunner.ID), models.ErrOrganizationRunnerCapacity)
	completed, err = second.Complete(f.db, secondRunner, "second", datatypes.JSON([]byte(`{}`)), 0, "", false, time.Now())
	require.NoError(t, err)
	require.True(t, completed)
	require.NoError(t, queued.Reserve(f.db, nextRunner.ID))

	additional := f.task(f.organizationID, fleet.ID)
	additionalRunner := f.runner(fleet.ID)
	require.ErrorIs(t, additional.Reserve(f.db, additionalRunner.ID), models.ErrOrganizationRunnerCapacity)
	f.plan.ParallelTaskLimit = new(2)
	require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
	require.NoError(t, additional.Reserve(f.db, additionalRunner.ID))

	f.plan.ParallelTaskLimit = nil
	require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
	unlimited := f.task(f.organizationID, fleet.ID)
	require.NoError(t, unlimited.Reserve(f.db, f.runner(fleet.ID).ID))
}

func TestRunnerCapacityUnlimitedAndBillingDisabled(t *testing.T) {
	for _, mode := range []string{"unlimited plan", "billing disabled"} {
		t.Run(mode, func(t *testing.T) {
			limit := new(1)
			if mode == "unlimited plan" {
				limit = nil
			}
			f := newRunnerCapacityTest(t, limit)
			if mode == "billing disabled" {
				t.Setenv(billingmodels.BillingEnabledEnv, "no")
			}
			fleet := f.fleet("unlimited-amd64")
			for range 3 {
				task := f.task(f.organizationID, fleet.ID)
				runner := f.runner(fleet.ID)
				require.NoError(t, task.Reserve(f.db, runner.ID))
				require.NoError(t, task.Start(f.db, runner, "test-store", time.Now()))
			}
		})
	}
}

func TestRunnerCapacityUsesTheEffectivePlan(t *testing.T) {
	for _, change := range []string{"missing plan", "expired trial"} {
		t.Run(change, func(t *testing.T) {
			f := newRunnerCapacityTest(t, new(1))
			fleet := f.fleet("effective-amd64")
			occupied := f.task(f.organizationID, fleet.ID)
			require.NoError(t, occupied.Reserve(f.db, f.runner(fleet.ID).ID))
			if change == "missing plan" {
				require.NoError(t, f.db.Where("organization_id = ?", f.organizationID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
			} else {
				f.plan.Category = "trial"
				require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
				_, err := billingmodels.SetAdminOrganizationPlan(f.db, f.organizationID, f.plan.Key, nil)
				require.NoError(t, err)
				require.NoError(t, f.db.Model(&billingmodels.OrganizationBillingPlan{}).
					Where("organization_id = ?", f.organizationID).
					Update("trial_ends_at", time.Now().Add(-time.Minute)).Error)
			}
			queued := f.task(f.organizationID, fleet.ID)
			require.NoError(t, queued.Reserve(f.db, f.runner(fleet.ID).ID))
			plan, err := billingmodels.FindOrganizationBillingPlan(f.db, f.organizationID)
			require.NoError(t, err)
			require.NotNil(t, plan)
			assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
		})
	}
}

func TestRunnerCapacityIgnoresLegacyJobsAndUnassignedRunners(t *testing.T) {
	f := newRunnerCapacityTest(t, new(1))
	fleet := f.fleet("integrated-amd64")
	legacy := f.task(f.organizationID, fleet.ID)
	require.NoError(t, f.db.Model(legacy).Updates(map[string]any{
		"backend": models.RunnerTaskBackendLegacy,
		"state":   models.RunnerTaskStateRunning,
	}).Error)
	for range 3 {
		runner := f.runner(fleet.ID)
		require.NoError(t, f.db.Model(runner).Update("state", models.RunnerStatePending).Error)
		f.runner(fleet.ID)
	}
	task := f.task(f.organizationID, fleet.ID)
	require.NoError(t, task.Reserve(f.db, f.runner(fleet.ID).ID))
	queued := f.task(f.organizationID, fleet.ID)
	require.ErrorIs(t, queued.Reserve(f.db, f.runner(fleet.ID).ID), models.ErrOrganizationRunnerCapacity)
}

type runnerCapacityTest struct {
	t              *testing.T
	db             *gorm.DB
	resource       *support.ResourceRegistry
	organizationID uuid.UUID
	plan           *billingmodels.BillingPlan
	queuedAt       time.Time
}

func newRunnerCapacityTest(t *testing.T, limit *int) *runnerCapacityTest {
	t.Helper()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	resource := support.Setup(t)
	f := &runnerCapacityTest{
		t:              t,
		db:             database.DB(t.Context()),
		resource:       resource,
		organizationID: resource.Organization.ID,
		queuedAt:       time.Now(),
	}
	f.plan = &billingmodels.BillingPlan{
		Key:               "runner_" + uuid.NewString()[:8],
		Name:              "Runner capacity",
		Category:          "paid",
		Visibility:        "private",
		CreditFrequency:   "none",
		CreditExpiration:  "never",
		ParallelTaskLimit: limit,
	}
	require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
	_, err := billingmodels.SetAdminOrganizationPlan(f.db, f.organizationID, f.plan.Key, nil)
	require.NoError(t, err)
	return f
}

func (f *runnerCapacityTest) fleet(slug string) *models.RunnerFleet {
	f.t.Helper()
	fleet := newTestRunnerFleet()
	fleet.Slug = slug
	require.NoError(f.t, fleet.Create(f.db))
	return fleet
}

func (f *runnerCapacityTest) runner(fleetID uuid.UUID) *models.Runner {
	f.t.Helper()
	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleetID,
		State:         models.RunnerStateIdle,
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(f.t, f.db.Create(runner).Error)
	return runner
}

func (f *runnerCapacityTest) task(organizationID, fleetID uuid.UUID) *models.RunnerTask {
	f.t.Helper()
	f.queuedAt = f.queuedAt.Add(time.Millisecond)
	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    organizationID,
		FleetID:           fleetID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          f.queuedAt,
		CreatedAt:         f.queuedAt,
		UpdatedAt:         f.queuedAt,
	}
	require.NoError(f.t, f.db.Create(task).Error)
	return task
}
