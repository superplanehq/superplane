package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type runnerMetricSpy struct {
	occupancy []runnerOccupancySample
	queueWait []runnerQueueWaitSample
	run       []runnerRunSample
}

type runnerOccupancySample struct {
	slug           string
	organizationID string
	state          string
	duration       time.Duration
}

type runnerQueueWaitSample struct {
	slug           string
	organizationID string
	outcome        string
	duration       time.Duration
}

type runnerRunSample struct {
	slug           string
	organizationID string
	state          string
	duration       time.Duration
}

func (s *runnerMetricSpy) install(t *testing.T) {
	t.Helper()
	previous := models.SetRunnerMetrics(models.RunnerMetrics{
		StateOccupancy: func(labels models.RunnerFleetLabels, state string, d time.Duration) {
			s.occupancy = append(s.occupancy, runnerOccupancySample{
				slug:           labels.FleetSlug,
				organizationID: labels.OrganizationID,
				state:          state,
				duration:       d,
			})
		},
		TaskQueueWait: func(labels models.RunnerFleetLabels, outcome string, d time.Duration) {
			s.queueWait = append(s.queueWait, runnerQueueWaitSample{
				slug:           labels.FleetSlug,
				organizationID: labels.OrganizationID,
				outcome:        outcome,
				duration:       d,
			})
		},
		TaskRun: func(labels models.RunnerFleetLabels, state string, d time.Duration) {
			s.run = append(s.run, runnerRunSample{
				slug:           labels.FleetSlug,
				organizationID: labels.OrganizationID,
				state:          state,
				duration:       d,
			})
		},
	})
	t.Cleanup(func() {
		models.SetRunnerMetrics(previous)
	})
}

func TestRunnerMetricsFollowTaskLifecycle(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	spy := &runnerMetricSpy{}
	spy.install(t)

	registeredAt := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	createdAt := registeredAt.Add(-time.Hour)
	queuedAt := registeredAt
	startedAt := registeredAt.Add(30 * time.Second)
	finishedAt := startedAt.Add(time.Minute)

	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateIdle,
		RunnerVersion: "0.1.0",
		RegisteredAt:  &registeredAt,
		CreatedAt:     createdAt,
		UpdatedAt:     startedAt,
	}
	require.NoError(t, db.Create(runner).Error)

	task := newMetricRunnerTask(resource.Organization.ID, fleet.ID, queuedAt)
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))
	require.Empty(t, spy.queueWait)
	require.NoError(t, task.Start(db, runner, "test-store", startedAt))

	require.Equal(t, []runnerOccupancySample{{
		slug:     fleet.Slug,
		state:    models.RunnerStateIdle,
		duration: 30 * time.Second,
	}}, spy.occupancy)
	require.Equal(t, []runnerQueueWaitSample{{
		slug:     fleet.Slug,
		outcome:  models.RunnerQueueWaitStarted,
		duration: 30 * time.Second,
	}}, spy.queueWait)

	require.NoError(t, task.Complete(
		db,
		runner,
		"completion-hash",
		datatypes.JSON([]byte(`{}`)),
		0,
		"",
		false,
		finishedAt,
	))
	require.NoError(t, task.Complete(
		db,
		runner,
		"completion-hash",
		datatypes.JSON([]byte(`{}`)),
		0,
		"",
		false,
		finishedAt.Add(time.Second),
	))

	require.Equal(t, []runnerOccupancySample{
		{slug: fleet.Slug, state: models.RunnerStateIdle, duration: 30 * time.Second},
		{slug: fleet.Slug, state: models.RunnerStateBusy, duration: time.Minute},
	}, spy.occupancy)
	require.Equal(t, []runnerRunSample{{
		slug:     fleet.Slug,
		state:    models.RunnerTaskStateSucceeded,
		duration: time.Minute,
	}}, spy.run)

	secondQueuedAt := finishedAt
	secondStartedAt := finishedAt.Add(10 * time.Second)
	secondTask := newMetricRunnerTask(resource.Organization.ID, fleet.ID, secondQueuedAt)
	require.NoError(t, db.Create(secondTask).Error)
	require.NoError(t, secondTask.Reserve(db, runner.ID))
	require.NoError(t, secondTask.Start(db, runner, "test-store", secondStartedAt))

	require.Equal(t, 10*time.Second, spy.occupancy[len(spy.occupancy)-1].duration)
	require.Equal(t, models.RunnerStateIdle, spy.occupancy[len(spy.occupancy)-1].state)
	require.Equal(t, 10*time.Second, spy.queueWait[len(spy.queueWait)-1].duration)
}

func TestCanceledQueuedTaskRecordsQueueWait(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	spy := &runnerMetricSpy{}
	spy.install(t)

	queuedAt := time.Now().Add(-15 * time.Second).Truncate(time.Millisecond)
	task := newMetricRunnerTask(resource.Organization.ID, fleet.ID, queuedAt)
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.RequestCancel(db, queuedAt.Add(15*time.Second)))

	require.Equal(t, []runnerQueueWaitSample{{
		slug:     fleet.Slug,
		outcome:  models.RunnerQueueWaitCanceled,
		duration: 15 * time.Second,
	}}, spy.queueWait)
	require.Empty(t, spy.run)
	require.Empty(t, spy.occupancy)
}

func TestLostRunningTaskRecordsBusyAndRunDuration(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	spy := &runnerMetricSpy{}
	spy.install(t)

	registeredAt := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	startedAt := registeredAt.Add(20 * time.Second)
	lostAt := startedAt.Add(40 * time.Second)
	lastSeenAt := lostAt.Add(-2 * time.Minute)
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateIdle,
		RunnerVersion: "0.1.0",
		RegisteredAt:  &registeredAt,
		LastSeenAt:    &lastSeenAt,
		CreatedAt:     registeredAt,
		UpdatedAt:     registeredAt,
	}
	require.NoError(t, db.Create(runner).Error)
	task := newMetricRunnerTask(resource.Organization.ID, fleet.ID, registeredAt)
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))
	require.NoError(t, task.Start(db, runner, "test-store", startedAt))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		return locked.MarkLost(tx, lostAt.Add(-time.Minute), lostAt)
	}))

	require.Equal(t, models.RunnerStateBusy, spy.occupancy[len(spy.occupancy)-1].state)
	require.Equal(t, 40*time.Second, spy.occupancy[len(spy.occupancy)-1].duration)
	require.Equal(t, []runnerRunSample{{
		slug:     fleet.Slug,
		state:    models.RunnerTaskStateLost,
		duration: 40 * time.Second,
	}}, spy.run)
}

func TestLostReservedTaskRecordsIdleAndQueueWait(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	spy := &runnerMetricSpy{}
	spy.install(t)

	registeredAt := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	lostAt := registeredAt.Add(time.Minute)
	lastSeenAt := lostAt.Add(-2 * time.Minute)
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateIdle,
		RunnerVersion: "0.1.0",
		RegisteredAt:  &registeredAt,
		LastSeenAt:    &lastSeenAt,
		CreatedAt:     registeredAt,
		UpdatedAt:     lostAt,
	}
	require.NoError(t, db.Create(runner).Error)
	task := newMetricRunnerTask(resource.Organization.ID, fleet.ID, registeredAt)
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		return locked.MarkLost(tx, lostAt.Add(-time.Minute), lostAt)
	}))

	require.Equal(t, []runnerOccupancySample{{
		slug:     fleet.Slug,
		state:    models.RunnerStateIdle,
		duration: time.Minute,
	}}, spy.occupancy)
	require.Equal(t, []runnerQueueWaitSample{{
		slug:     fleet.Slug,
		outcome:  models.RunnerQueueWaitLost,
		duration: time.Minute,
	}}, spy.queueWait)
	require.Empty(t, spy.run)
}

func TestTerminateIdleRunnerRecordsIdleDuration(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	spy := &runnerMetricSpy{}
	spy.install(t)

	registeredAt := time.Now().Add(-2 * time.Minute)
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateIdle,
		RunnerVersion: "0.1.0",
		RegisteredAt:  &registeredAt,
		CreatedAt:     registeredAt.Add(-time.Hour),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(runner).Error)
	require.NoError(t, runner.Terminate(db, models.RunnerTerminationRequested))

	require.Len(t, spy.occupancy, 1)
	assert.Equal(t, models.RunnerStateIdle, spy.occupancy[0].state)
	assert.Equal(t, fleet.Slug, spy.occupancy[0].slug)
	assert.InDelta(t, (2 * time.Minute).Seconds(), spy.occupancy[0].duration.Seconds(), 2)
	require.Empty(t, spy.run)
	require.Empty(t, spy.queueWait)

	pending := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStatePending,
		RunnerVersion: "0.1.0",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(pending).Error)
	require.NoError(t, pending.Terminate(db, models.RunnerTerminationRequested))
	require.Len(t, spy.occupancy, 1)
}

func newMetricRunnerTask(organizationID, fleetID uuid.UUID, queuedAt time.Time) *models.RunnerTask {
	return &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    organizationID,
		FleetID:           fleetID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          queuedAt,
		CreatedAt:         queuedAt,
		UpdatedAt:         queuedAt,
	}
}
