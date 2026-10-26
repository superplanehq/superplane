package models_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// These tests own organization reservation accounting. Concurrent direct and
// automatic reservations must share the same durable job-state boundary.
func TestRunnerReservationsCountEachJobAndRetainAssignedCancellation(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	t.Setenv(models.BillingEnabledEnv, "yes")
	require.NoError(t, db.Model(&models.BillingPlan{}).Where("key = ?", "trial").Update("parallel_task_limit", 4).Error)
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	factory, err := models.CreateFactory(db, r.Organization.ID, "Runner slot workspace", "", "RSLOT")
	require.NoError(t, err)
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Runner slot automation", "start")
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "One task with five runner jobs", "", &r.User, nil, &run.ID)
	require.NoError(t, err)
	event := support.EmitCanvasEventForNode(t, canvas.ID, entrypoint, "default", nil)
	var jobs []*models.RunnerTask
	for range 5 {
		job, runner := createReservationFixture(t, db, r.Organization.ID, fleet.ID)
		jobs = append(jobs, job)
		execution := &models.CanvasNodeExecution{ID: uuid.New(), WorkflowID: canvas.ID, NodeID: entrypoint,
			RunID: run.ID, RootEventID: event.ID, EventID: event.ID, State: models.CanvasNodeExecutionStatePending,
			Configuration: datatypes.NewJSONType(map[string]any{})}
		require.NoError(t, db.Create(execution).Error)
		require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, entrypoint, execution.ID, "task_id", job.ID.String()))
		if len(jobs) == 5 {
			require.ErrorIs(t, job.Reserve(db, runner.ID), models.ErrOrganizationRunnerCapacity)
			continue
		}
		require.NoError(t, job.Reserve(db, runner.ID))
		require.NoError(t, job.Reserve(db, runner.ID))
	}
	count, err := models.CountActiveOrganizationRunnerJobs(db, r.Organization.ID)
	require.NoError(t, err)
	require.EqualValues(t, 4, count)
	queued, err := models.ListOrganizationQueuedRunnerJobs(db, r.Organization.ID)
	require.NoError(t, err)
	require.Len(t, queued, 1)
	require.Equal(t, order.ID, queued[0].WorkOrderID)
	// A dispatched cancellation remains charged until the runner confirms stop.
	now := time.Now()
	require.NoError(t, db.Model(jobs[0]).Update("dispatch_started_at", now).Error)
	require.NoError(t, jobs[0].RequestCancel(db, now))
	count, err = models.CountActiveOrganizationRunnerJobs(db, r.Organization.ID)
	require.NoError(t, err)
	require.EqualValues(t, 4, count)
	// A certainly-unsent reservation is safe to cancel immediately.
	require.NoError(t, jobs[1].RequestCancel(db, now))
	count, err = models.CountActiveOrganizationRunnerJobs(db, r.Organization.ID)
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
	require.NoError(t, db.Model(&models.BillingPlan{}).Where("key = ?", "trial").Update("parallel_task_limit", 2).Error)
	last, err := models.FindRunnerTask(db, jobs[4].ID)
	require.NoError(t, err)
	_, runner := createReservationFixture(t, db, r.Organization.ID, fleet.ID)
	require.ErrorIs(t, last.Reserve(db, runner.ID), models.ErrOrganizationRunnerCapacity)
	first, err := models.FindRunnerTask(db, jobs[0].ID)
	require.NoError(t, err)
	require.Equal(t, models.RunnerTaskStateReserved, first.State)
	assigned, err := models.FindRunner(db, *first.RunnerID)
	require.NoError(t, err)
	require.NoError(t, first.Start(db, assigned, "test-store", now))
	_, err = first.Complete(db, assigned, "confirmed-cancel", datatypes.JSON([]byte(`{}`)), 0, "", true, now)
	require.NoError(t, err)
	require.ErrorIs(t, last.Reserve(db, runner.ID), models.ErrOrganizationRunnerCapacity)
	require.NoError(t, jobs[2].RequestCancel(db, now))
	require.NoError(t, last.Reserve(db, runner.ID))
}

func TestConcurrentRunnerReservationsAcrossFleetsShareOrganizationCapacity(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	t.Setenv(models.BillingEnabledEnv, "yes")
	require.NoError(t, db.Model(&models.BillingPlan{}).Where("key = ?", "trial").Update("parallel_task_limit", 2).Error)
	fleets := []*models.RunnerFleet{newTestRunnerFleet(), newTestRunnerFleet()}
	fleets[1].Slug += "-other"
	for _, fleet := range fleets {
		require.NoError(t, fleet.Create(db))
	}
	type reservation struct {
		job    *models.RunnerTask
		runner *models.Runner
	}
	var candidates []reservation
	for index := range 12 {
		job, runner := createReservationFixture(t, db, r.Organization.ID, fleets[index%2].ID)
		candidates = append(candidates, reservation{job, runner})
	}
	ready := make(chan struct{})
	results := make(chan error, len(candidates))
	var group sync.WaitGroup
	for _, candidate := range candidates {
		group.Add(1)
		go func() { defer group.Done(); <-ready; results <- candidate.job.Reserve(db, candidate.runner.ID) }()
	}
	close(ready)
	group.Wait()
	close(results)
	admitted := 0
	for err := range results {
		if err == nil {
			admitted++
			continue
		}
		require.ErrorIs(t, err, models.ErrOrganizationRunnerCapacity)
	}
	require.Equal(t, 2, admitted)
	other, err := models.CreateOrganization("Other runner organization", "")
	require.NoError(t, err)
	job, runner := createReservationFixture(t, db, other.ID, fleets[0].ID)
	require.NoError(t, job.Reserve(db, runner.ID))
}

func TestAutomaticReservationSkipsLockedOrganizationsWithoutCandidateLimit(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	t.Setenv(models.BillingEnabledEnv, "no")
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	for range 105 {
		createReservationFixture(t, db, r.Organization.ID, fleet.ID)
	}
	other, err := models.CreateOrganization("Available runner organization", "")
	require.NoError(t, err)
	eligible, runner := createReservationFixture(t, db, other.ID, fleet.ID)
	locked := db.Begin()
	require.NoError(t, locked.Error)
	t.Cleanup(func() { _ = locked.Rollback().Error })
	require.NoError(t, models.LockOrganizationRunnerAdmission(locked, r.Organization.ID))
	reserved, err := runner.ReserveNextTask(db)
	require.NoError(t, err)
	require.Equal(t, eligible.ID, reserved.ID)
}

func TestSkippedReservationReleasesCandidateLocks(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	t.Setenv(models.BillingEnabledEnv, "yes")
	require.NoError(t, db.Model(&models.BillingPlan{}).Where("key = ?", "trial").Update("parallel_task_limit", 1).Error)
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	active, activeRunner := createReservationFixture(t, db, r.Organization.ID, fleet.ID)
	require.NoError(t, active.Reserve(db, activeRunner.ID))
	blocked, runner := createReservationFixture(t, db, r.Organization.ID, fleet.ID)
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := runner.ReserveNextTask(tx)
		require.ErrorIs(t, err, models.ErrRunnerTaskNotFound)
		completed := make(chan error, 1)
		go func() {
			completed <- db.Transaction(func(other *gorm.DB) error {
				if err := other.Exec("SET LOCAL lock_timeout = '1s'").Error; err != nil {
					return err
				}
				return blocked.RequestCancel(other, time.Now())
			})
		}()
		return <-completed
	})
	require.NoError(t, err)
}

func createReservationFixture(t *testing.T, db *gorm.DB, organizationID, fleetID uuid.UUID) (*models.RunnerTask, *models.Runner) {
	t.Helper()
	now := time.Now()
	job := &models.RunnerTask{ID: uuid.New(), OrganizationID: organizationID, FleetID: fleetID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("encrypted"), QueuedAt: now, CreatedAt: now, UpdatedAt: now}
	runner := &models.Runner{ID: uuid.New(), FleetID: fleetID, State: models.RunnerStateIdle,
		RunnerVersion: "dev", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(job).Error)
	require.NoError(t, db.Create(runner).Error)
	return job, runner
}
