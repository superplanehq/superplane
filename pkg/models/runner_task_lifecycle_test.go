package models_test

import (
	"os"
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
	assert.Equal(t, []byte("ciphertext"), task.PayloadCiphertext)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.PayloadCiphertext, stored.PayloadCiphertext)
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
	assert.Nil(t, lostTask.PayloadCiphertext)
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
	assert.Nil(t, reloadedTask.PayloadCiphertext)
	lifecycle, err := reloadedTask.FindLifecycle(db)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskLogStateArchivable, lifecycle.State)
}

func TestRequestCancelRunnerTasksUsesCurrentTaskState(t *testing.T) {
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
	finished := newTask()
	queued := newTask()
	require.NoError(t, db.Create(finished).Error)
	require.NoError(t, db.Create(queued).Error)
	require.NoError(t, finished.Reserve(db, runner.ID))
	require.NoError(t, finished.Start(db, runner, "test-store", now))

	stale := []models.RunnerTask{*finished, *queued}
	completed, err := finished.Complete(db, runner, "done", datatypes.JSON([]byte(`{}`)), 0, "", false, now)
	require.NoError(t, err)
	require.True(t, completed)

	current, err := models.RequestCancelRunnerTasks(db, stale, now)
	require.NoError(t, err)
	require.Len(t, current, 2)
	assert.Equal(t, models.RunnerTaskStateSucceeded, current[0].State)
	assert.Nil(t, current[0].CancelRequestedAt)
	assert.Equal(t, models.RunnerTaskStateCanceled, current[1].State)
	assert.Nil(t, current[0].PayloadCiphertext)
	assert.Nil(t, current[1].PayloadCiphertext)
	reloaded, err := models.FindRunnerTask(db, finished.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateSucceeded, reloaded.State)
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
	assert.Nil(t, firstTask.PayloadCiphertext)
	stored, err := models.FindRunnerTask(db, firstTask.ID)
	require.NoError(t, err)
	assert.Nil(t, stored.PayloadCiphertext)
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

// Each case exercises a distinct state writer, including the conditional bulk
// update that releases a reservation when a runner terminates.
func TestRunnerTaskPayloadLifetime(t *testing.T) {
	for _, scenario := range []string{"cancel pending", "cancel queued", "fail pending", "fail queued", "request reserved", "request running", "terminate canceled", "terminate requeue", "lose reserved", "complete failed", "complete canceled"} {
		t.Run(scenario, func(t *testing.T) {
			require.NoError(t, database.TruncateTables())
			resource := support.Setup(t)
			db := database.DB(t.Context())
			fleet := newTestRunnerFleet()
			require.NoError(t, fleet.Create(db))
			now := time.Now()
			lastSeen := now.Add(-2 * time.Minute)
			runner := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle, LastSeenAt: &lastSeen}
			require.NoError(t, db.Create(runner).Error)
			task := &models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("secret"), QueuedAt: now}
			if scenario == "cancel pending" || scenario == "fail pending" {
				task.State = models.RunnerTaskStatePending
			}
			require.NoError(t, db.Create(task).Error)
			expected := models.RunnerTaskStateCanceled
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				switch scenario {
				case "cancel pending", "cancel queued":
					return task.RequestCancel(tx, now)
				case "fail pending", "fail queued":
					expected = models.RunnerTaskStateFailed
					return task.FailUnassigned(tx, "admission denied", now)
				}
				if err := task.Reserve(tx, runner.ID); err != nil {
					return err
				}
				if scenario == "request running" || scenario == "complete failed" || scenario == "complete canceled" {
					if err := task.Start(tx, runner, "test-store", now); err != nil {
						return err
					}
				}
				switch scenario {
				case "request reserved", "request running":
					expected = task.State
					if err := task.RequestCancel(tx, now); err != nil {
						return err
					}
					require.NotNil(t, task.CancelRequestedAt)
					return nil
				case "terminate canceled":
					if err := task.RequestCancel(tx, now); err != nil {
						return err
					}
					return runner.Terminate(tx, models.RunnerTerminationRequested)
				case "terminate requeue":
					expected = models.RunnerTaskStateQueued
					return runner.Terminate(tx, models.RunnerTerminationRequested)
				case "lose reserved":
					expected = models.RunnerTaskStateLost
					_, err := runner.MarkLost(tx, now.Add(-time.Minute), now)
					return err
				default:
					canceled := scenario == "complete canceled"
					if !canceled {
						expected = models.RunnerTaskStateFailed
					}
					_, err := task.Complete(tx, runner, "hash", datatypes.JSON([]byte(`{"answer":42}`)), 1, "failure", canceled, now)
					return err
				}
			}))
			stored, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			require.Equal(t, expected, stored.State)
			if stored.IsTerminal() {
				assert.Nil(t, stored.PayloadCiphertext)
			} else {
				assert.Equal(t, []byte("secret"), stored.PayloadCiphertext)
			}
			// Bulk updates have no task receiver; all other writers must also update it.
			if scenario != "terminate canceled" && scenario != "terminate requeue" && scenario != "lose reserved" {
				assert.Equal(t, stored.PayloadCiphertext, task.PayloadCiphertext)
			}
		})
	}
}

func TestRunnerTaskPayloadConstraint(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	t.Setenv("ENCRYPTION_KEY", "")
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	for _, state := range []string{models.RunnerTaskStatePending, models.RunnerTaskStateQueued, models.RunnerTaskStateReserved, models.RunnerTaskStateRunning, models.RunnerTaskStateSucceeded, models.RunnerTaskStateFailed, models.RunnerTaskStateCanceled, models.RunnerTaskStateLost} {
		t.Run(state, func(t *testing.T) {
			task := &models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendLegacy, State: state, QueuedAt: time.Now()}
			err := db.Create(task).Error
			if task.IsTerminal() {
				require.NoError(t, err)
				stored, err := models.FindRunnerTask(db, task.ID)
				require.NoError(t, err)
				assert.Nil(t, stored.PayloadCiphertext)
				// Reopening an imported terminal task requires a payload too.
				require.ErrorContains(t, db.Model(task).Update("state", models.RunnerTaskStateQueued).Error, "runner_tasks_nonterminal_payload")
				return
			}
			require.ErrorContains(t, err, "runner_tasks_nonterminal_payload")
			task.PayloadCiphertext = []byte("secret")
			require.NoError(t, db.Create(task).Error)
			require.ErrorContains(t, db.Model(task).Update("payload_ciphertext", nil).Error, "runner_tasks_nonterminal_payload")
		})
	}
}

func TestTerminalRunnerTaskPayloadCleanup(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	now := time.Now().Truncate(time.Microsecond)
	states := []string{models.RunnerTaskStateSucceeded, models.RunnerTaskStateFailed, models.RunnerTaskStateCanceled, models.RunnerTaskStateLost}
	tasks := make([]models.RunnerTask, 1001)
	for i := range tasks {
		tasks[i] = models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendLegacy, State: states[i%len(states)], PayloadCiphertext: []byte("secret"), QueuedAt: now, FinishedAt: &now, UpdatedAt: now, Result: datatypes.JSON([]byte(`{"answer":42}`))}
	}
	require.NoError(t, db.CreateInBatches(&tasks, 100).Error)
	for _, state := range []string{models.RunnerTaskStatePending, models.RunnerTaskStateQueued, models.RunnerTaskStateReserved, models.RunnerTaskStateRunning} {
		task := models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: state, PayloadCiphertext: []byte("active"), QueuedAt: now}
		require.NoError(t, db.Create(&task).Error)
	}
	imported := models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendLegacy, State: models.RunnerTaskStateSucceeded, QueuedAt: now}
	require.NoError(t, db.Create(&imported).Error)
	// Rotation may read ciphertext before a concurrent cleanup removes it.
	page, err := models.ReadRunnerTaskPayloadPage(db, "", 2000)
	require.NoError(t, err)
	require.Len(t, page, 1005)
	cleanup, err := os.ReadFile("../../scripts/clear_terminal_runner_task_payloads.sql")
	require.NoError(t, err)
	locked := db.Begin()
	require.NoError(t, locked.Error)
	t.Cleanup(func() { locked.Rollback() })
	require.NoError(t, locked.Exec("SELECT id FROM runner_tasks WHERE id = ? FOR UPDATE", tasks[0].ID).Error)
	for _, count := range []int64{1000, 0} {
		result := db.Exec(string(cleanup))
		require.NoError(t, result.Error)
		assert.Equal(t, count, result.RowsAffected)
	}
	require.NoError(t, locked.Rollback().Error)
	for _, count := range []int64{1, 0} {
		result := db.Exec(string(cleanup))
		require.NoError(t, result.Error)
		assert.Equal(t, count, result.RowsAffected)
	}
	var active []models.RunnerTask
	require.NoError(t, db.Where("payload_ciphertext IS NOT NULL").Find(&active).Error)
	require.Len(t, active, 4)
	for _, task := range active {
		assert.Equal(t, []byte("active"), task.PayloadCiphertext)
	}
	stored, err := models.FindRunnerTask(db, tasks[0].ID)
	require.NoError(t, err)
	assert.Nil(t, stored.PayloadCiphertext)
	assert.Equal(t, tasks[0].State, stored.State)
	assert.JSONEq(t, `{"answer":42}`, string(stored.Result))
	assert.WithinDuration(t, now, stored.UpdatedAt, time.Microsecond)
	require.NotNil(t, stored.FinishedAt)
	assert.WithinDuration(t, now, *stored.FinishedAt, time.Microsecond)
	pageAfter, err := models.ReadRunnerTaskPayloadPage(db, "", 2000)
	require.NoError(t, err)
	require.Len(t, pageAfter, 4)
	for _, row := range page {
		if row.Cursor != stored.ID.String() {
			continue
		}
		replaced, err := models.ReplaceRunnerTaskPayload(db, row, []byte("rotated"))
		require.NoError(t, err)
		assert.False(t, replaced, "rotation must not restore a discarded payload")
	}
}
