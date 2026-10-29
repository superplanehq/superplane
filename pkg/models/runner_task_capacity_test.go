package models_test

import (
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

func TestRunnerSchedulingSerializesOrganizationsAcrossFleets(t *testing.T) {
	f := newRunnerCapacityTest(t, new(2))
	first := f.fleet("first-amd64")
	second := f.fleet("second-amd64")
	var tasks []*models.RunnerTask
	for i := range 12 {
		fleet := first
		if i%2 != 0 {
			fleet = second
		}
		tasks = append(tasks, f.task(f.organizationID, fleet.ID))
	}
	start := make(chan struct{})
	errors := make([]error, 8)
	var group sync.WaitGroup
	for i := range errors {
		group.Go(func() {
			<-start
			_, errors[i] = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
		})
	}
	close(start)
	group.Wait()
	for _, err := range errors {
		require.NoError(t, err)
	}
	for i, task := range tasks {
		stored, err := models.FindRunnerTask(f.db, task.ID)
		require.NoError(t, err)
		expected := models.RunnerTaskStatePending
		if i < 2 {
			expected = models.RunnerTaskStateQueued
			require.NotNil(t, stored.ReadyAt)
		}
		assert.Equal(t, expected, stored.State)
	}
	// Assignment keeps the slot and cannot release another pending task.
	reserved, err := f.runner(first.ID).ReserveNextTask(f.db)
	require.NoError(t, err)
	assert.Equal(t, tasks[0].ID, reserved.Task.ID)
	require.NoError(t, reserved.Task.Reserve(f.db, *reserved.Task.RunnerID))
	scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
	require.NoError(t, err)
	assert.Empty(t, scheduled)
}

func TestRunnerReservationCannotClaimPendingTasks(t *testing.T) {
	f := newRunnerCapacityTest(t, new(1))
	fleet := f.fleet("pending-amd64")
	task := f.task(f.organizationID, fleet.ID)
	runner := f.runner(fleet.ID)
	require.ErrorIs(t, task.Reserve(f.db, runner.ID), models.ErrRunnerTaskNotReservable)
	reservation, err := runner.ReserveNextTask(f.db)
	require.NoError(t, err)
	assert.Nil(t, reservation.Task)
}

func TestRunnerSchedulingReleasesSlotsThroughLifecycle(t *testing.T) {
	for _, release := range []string{"pending cancellation", "queued cancellation", "completion", "running cancellation", "reserved runner termination", "reserved connection loss", "running runner loss"} {
		t.Run(release, func(t *testing.T) {
			f := newRunnerCapacityTest(t, new(1))
			fleet := f.fleet("lifecycle-amd64")
			first := f.task(f.organizationID, fleet.ID)
			next := f.task(f.organizationID, fleet.ID)
			if release == "pending cancellation" {
				require.NoError(t, first.RequestCancel(f.db, time.Now()))
			} else {
				scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
				require.NoError(t, err)
				require.Len(t, scheduled, 1)
				first = &scheduled[0]
				if release == "queued cancellation" {
					require.NoError(t, first.RequestCancel(f.db, time.Now()))
				} else {
					runner := f.runner(fleet.ID)
					require.NoError(t, first.Reserve(f.db, runner.ID))
					if release != "reserved runner termination" && release != "reserved connection loss" {
						require.NoError(t, first.Start(f.db, runner, "test-store", time.Now()))
					}
					if release == "running cancellation" {
						require.NoError(t, first.RequestCancel(f.db, time.Now()))
						scheduled, err = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
						require.NoError(t, err)
						assert.Empty(t, scheduled, "a cancellation request must retain its slot")
					}
					if release == "completion" || release == "running cancellation" {
						_, err := first.Complete(f.db, runner, "done", datatypes.JSON([]byte(`{}`)), 0, "", release == "running cancellation", time.Now())
						require.NoError(t, err)
					} else if release == "reserved runner termination" {
						require.NoError(t, runner.Terminate(f.db, models.RunnerTerminationProvisioningFailed))
					} else {
						now := time.Now()
						require.NoError(t, f.db.Model(runner).Update("last_seen_at", now.Add(-2*time.Minute)).Error)
						require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
							locked, err := models.LockRunner(tx, runner.ID)
							if err != nil {
								return err
							}
							_, err = locked.MarkLost(tx, now.Add(-time.Minute), now)
							return err
						}))
					}
				}
			}
			scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
			require.NoError(t, err)
			if release == "reserved runner termination" {
				assert.Empty(t, scheduled, "returning a reservation to queued keeps its slot")
				stored, err := models.FindRunnerTask(f.db, first.ID)
				require.NoError(t, err)
				assert.Equal(t, models.RunnerTaskStateQueued, stored.State)
				return
			}
			require.Len(t, scheduled, 1)
			assert.Equal(t, next.ID, scheduled[0].ID)
		})
	}
}

func TestRunnerSchedulingReadsPlanChangesWithoutRevokingReadyTasks(t *testing.T) {
	f := newRunnerCapacityTest(t, new(2))
	fleet := f.fleet("plan-amd64")
	for range 4 {
		f.task(f.organizationID, fleet.ID)
	}
	scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
	require.NoError(t, err)
	require.Len(t, scheduled, 2)
	f.plan.ParallelTaskLimit = new(1)
	require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
	remaining, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
	require.NoError(t, err)
	assert.Empty(t, remaining)
	// A lower limit does not revoke a slot acquired before the plan change.
	require.NoError(t, scheduled[0].Reserve(f.db, f.runner(fleet.ID).ID))
	f.plan.ParallelTaskLimit = new(3)
	require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
	remaining, err = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
}

func TestRunnerSchedulingUnlimitedAndBillingDisabled(t *testing.T) {
	for _, mode := range []string{"unlimited plan", "billing disabled", "missing plan", "expired trial"} {
		t.Run(mode, func(t *testing.T) {
			limit := new(1)
			if mode == "unlimited plan" {
				limit = nil
			}
			f := newRunnerCapacityTest(t, limit)
			switch mode {
			case "billing disabled":
				t.Setenv(billingmodels.BillingEnabledEnv, "no")
			case "missing plan":
				require.NoError(t, f.db.Where("organization_id = ?", f.organizationID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
			case "expired trial":
				f.plan.Category = "trial"
				require.NoError(t, billingmodels.SaveBillingPlan(f.db, f.plan))
				_, err := billingmodels.SetAdminOrganizationPlan(f.db, f.organizationID, f.plan.Key, nil)
				require.NoError(t, err)
				require.NoError(t, f.db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", f.organizationID).Update("trial_ends_at", time.Now().Add(-time.Minute)).Error)
			}
			fleet := f.fleet("unlimited-amd64")
			for range 3 {
				f.task(f.organizationID, fleet.ID)
			}
			scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 2)
			require.NoError(t, err)
			assert.Len(t, scheduled, 2, "even unlimited organizations use bounded batches")
			scheduled, err = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 2)
			require.NoError(t, err)
			assert.Len(t, scheduled, 1)
		})
	}
}

func TestRunnerSchedulingSkipsUnavailableFleetsAndSeparatesOrganizations(t *testing.T) {
	f := newRunnerCapacityTest(t, new(1))
	disabled := f.fleet("disabled-amd64")
	fleet := f.fleet("enabled-amd64")
	require.NoError(t, disabled.SetEnabled(f.db, false))
	f.task(f.organizationID, disabled.ID)
	eligible := f.task(f.organizationID, fleet.ID)
	legacy := f.task(f.organizationID, fleet.ID)
	require.NoError(t, f.db.Model(legacy).Updates(map[string]any{"backend": models.RunnerTaskBackendLegacy, "state": models.RunnerTaskStateRunning}).Error)
	for range 3 {
		f.runner(fleet.ID)
	}
	scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
	require.NoError(t, err)
	require.Len(t, scheduled, 1)
	assert.Equal(t, eligible.ID, scheduled[0].ID)
	other := support.CreateOrganization(t, f.resource, f.resource.User)
	_, err = billingmodels.SetAdminOrganizationPlan(f.db, other.ID, f.plan.Key, nil)
	require.NoError(t, err)
	otherTask := f.task(other.ID, fleet.ID)
	scheduled, err = models.ScheduleOrganizationRunnerTasks(f.db, other.ID, 100)
	require.NoError(t, err)
	require.Len(t, scheduled, 1)
	assert.Equal(t, otherTask.ID, scheduled[0].ID)
}

func TestRunnerSchedulingSkipsLockedOrganizationsAndPaginatesPendingWork(t *testing.T) {
	f := newRunnerCapacityTest(t, new(1))
	fleet := f.fleet("shared-amd64")
	blocked := f.task(f.organizationID, fleet.ID)
	other := support.CreateOrganization(t, f.resource, f.resource.User)
	eligible := f.task(other.ID, fleet.ID)
	holder := f.db.Begin()
	require.NoError(t, holder.Error)
	defer holder.Rollback()
	require.NoError(t, holder.Model(&models.Organization{}).
		Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).First(&models.Organization{}, "id = ?", f.organizationID).Error)
	tasks, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
	require.NoError(t, err)
	assert.Empty(t, tasks)
	// Pagination includes full or locked organizations without hiding later work.
	page, err := models.ListPendingRunnerTaskOrganizations(f.db, uuid.Nil, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	next, err := models.ListPendingRunnerTaskOrganizations(f.db, page[0], 1)
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.ElementsMatch(t, []uuid.UUID{f.organizationID, other.ID}, []uuid.UUID{page[0], next[0]})
	tasks, err = models.ScheduleOrganizationRunnerTasks(f.db, other.ID, 100)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, eligible.ID, tasks[0].ID)
	stored, err := models.FindRunnerTask(f.db, blocked.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStatePending, stored.State)
}

func TestRunnerSchedulingKeepsOrReleasesSlotOnRegistrationExpiry(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		name := "reservation returns to queued with its slot"
		if canceled {
			name = "canceled reservation releases its slot"
		}
		t.Run(name, func(t *testing.T) {
			f := newRunnerCapacityTest(t, new(1))
			fleet := f.fleet("expiry-amd64")
			first := f.task(f.organizationID, fleet.ID)
			next := f.task(f.organizationID, fleet.ID)
			scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
			require.NoError(t, err)
			require.Len(t, scheduled, 1)
			assert.Equal(t, first.ID, scheduled[0].ID)
			first = &scheduled[0]
			runner := f.runner(fleet.ID)
			require.NoError(t, f.db.Model(runner).Update("state", models.RunnerStatePending).Error)
			require.NoError(t, first.Reserve(f.db, runner.ID))
			if canceled {
				require.NoError(t, first.RequestCancel(f.db, time.Now()))
				scheduled, err = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
				require.NoError(t, err)
				assert.Empty(t, scheduled, "a cancellation request retains the reservation's slot")
			}
			now := time.Now()
			require.NoError(t, f.db.Create(&models.RunnerRegistration{
				JTI: uuid.New(), RunnerID: runner.ID, ExpiresAt: now.Add(-time.Minute), CreatedAt: now.Add(-2 * time.Minute),
			}).Error)
			require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
				locked, err := models.LockRunner(tx, runner.ID)
				if err != nil {
					return err
				}
				return locked.Expire(tx, now)
			}))
			stored, err := models.FindRunnerTask(f.db, first.ID)
			require.NoError(t, err)
			assert.Nil(t, stored.RunnerID)
			assert.Nil(t, stored.ReservedAt)
			scheduled, err = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 100)
			require.NoError(t, err)
			if canceled {
				assert.Equal(t, models.RunnerTaskStateCanceled, stored.State)
				require.NotNil(t, stored.FinishedAt)
				require.Len(t, scheduled, 1)
				assert.Equal(t, next.ID, scheduled[0].ID)
				return
			}
			assert.Equal(t, models.RunnerTaskStateQueued, stored.State)
			assert.Nil(t, stored.FinishedAt)
			assert.Empty(t, scheduled)
		})
	}
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
		State:             models.RunnerTaskStatePending,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          f.queuedAt,
		CreatedAt:         f.queuedAt,
		UpdatedAt:         f.queuedAt,
	}
	require.NoError(f.t, f.db.Create(task).Error)
	return task
}
