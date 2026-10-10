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

func TestRunnerTaskStartCreatesLogLifecycle(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())

	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateIdle,
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)

	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))

	startedAt := time.Now()
	const activeLogStore = "test-store"
	require.NoError(t, task.Start(db, runner, activeLogStore, startedAt))

	assert.Equal(t, models.RunnerTaskStateRunning, task.State)
	assert.Equal(t, models.RunnerStateBusy, runner.State)
	lifecycle, err := task.FindLifecycle(db)
	require.NoError(t, err)
	assert.Equal(t, activeLogStore, lifecycle.ActiveStore)
	assert.Equal(t, models.RunnerTaskLogStateActive, lifecycle.State)
	assert.WithinDuration(t, startedAt, lifecycle.CreatedAt, time.Millisecond)
	assert.WithinDuration(t, startedAt, lifecycle.UpdatedAt, time.Millisecond)
}

func TestLostRunningTaskBecomesArchivable(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())

	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	now := time.Now()
	lastSeenAt := now.Add(-2 * time.Minute)
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateBusy,
		RunnerVersion: "0.1.0",
		LastSeenAt:    &lastSeenAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)

	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		RunnerID:          &runner.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateRunning,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          now,
		StartedAt:         &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, db.Create(&models.RunnerTaskLogLifecycle{
		TaskID:      task.ID,
		ActiveStore: "test-store",
		State:       models.RunnerTaskLogStateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}).Error)

	var lostTask *models.RunnerTask
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		lostTask, err = locked.MarkLost(tx, now.Add(-time.Minute), now)
		return err
	}))
	require.NotNil(t, lostTask)
	assert.Equal(t, task.ID, lostTask.ID)
	assert.Equal(t, models.RunnerTaskStateLost, lostTask.State)
	require.NotNil(t, lostTask.FinishedAt)
	assert.Equal(t, now, *lostTask.FinishedAt)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		lostTask, err = locked.MarkLost(tx, now.Add(-time.Minute), now)
		return err
	}))
	assert.Nil(t, lostTask)

	reloadedTask, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateLost, reloadedTask.State)
	lifecycle, err := reloadedTask.FindLifecycle(db)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskLogStateArchivable, lifecycle.State)
}

func TestReusableRunnerCanCompleteSequentialTasks(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())

	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateIdle,
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)

	newTask := func() *models.RunnerTask {
		return &models.RunnerTask{
			ID:                uuid.New(),
			OrganizationID:    resource.Organization.ID,
			FleetID:           fleet.ID,
			Backend:           models.RunnerTaskBackendIntegrated,
			State:             models.RunnerTaskStateQueued,
			PayloadCiphertext: []byte("ciphertext"),
			QueuedAt:          now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}
	firstTask := newTask()
	secondTask := newTask()
	require.NoError(t, db.Create(firstTask).Error)
	require.NoError(t, db.Create(secondTask).Error)

	require.NoError(t, firstTask.Reserve(db, runner.ID))
	require.NoError(t, firstTask.Start(db, runner, "test-store", now))
	completed, err := firstTask.Complete(
		db,
		runner,
		"first-completion",
		datatypes.JSON([]byte(`{}`)),
		0,
		"",
		false,
		now,
	)
	require.NoError(t, err)
	assert.True(t, completed)
	completed, err = firstTask.Complete(
		db,
		runner,
		"first-completion",
		datatypes.JSON([]byte(`{}`)),
		0,
		"",
		false,
		now,
	)
	require.NoError(t, err)
	assert.False(t, completed)

	require.NoError(t, secondTask.Reserve(db, runner.ID))
	assert.Equal(t, models.RunnerTaskStateReserved, secondTask.State)
}
