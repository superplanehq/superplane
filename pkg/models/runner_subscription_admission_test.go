package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestRunnerAdmissionRejectsInvalidSubscriptionAndContinuesFleetScan(t *testing.T) {
	for _, reason := range []string{"revoked lease", "disconnected connection", "missing connection", "missing owner", "unavailable model", "missing task association"} {
		t.Run(reason, func(t *testing.T) {
			db, connection, rejected, runner, order := queuedSubscriptionRunnerTask(t)
			expectedError := models.ErrCodexConnectionStale
			switch reason {
			case "revoked lease":
				require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", rejected.ID).Update("revoked_at", time.Now()).Error)
			case "disconnected connection":
				expectedError = models.ErrCodexConnectionNotReady
				require.NoError(t, db.Model(connection).Update("status", models.CodexConnectionStatusDisconnected).Error)
			case "missing connection":
				expectedError = models.ErrCodexConnectionNotFound
				execution, err := rejected.FindExecution(db)
				require.NoError(t, err)
				require.NoError(t, db.Model(execution).Update("configuration", []byte(`{"credentials":{"source":"subscription"}}`)).Error)
				require.NoError(t, db.Delete(connection).Error)
			case "missing owner":
				expectedError = models.ErrSubscriptionOwnerRequired
				require.NoError(t, db.Where("work_order_id = ?", order.ID).Delete(&models.FactoryWorkOrderAssignee{}).Error)
			case "unavailable model":
				expectedError = models.ErrSubscriptionModelUnavailable
				execution, err := rejected.FindExecution(db)
				require.NoError(t, err)
				require.NoError(t, db.Model(execution).Update("configuration", []byte(`{"model":"removed-model"}`)).Error)
			case "missing task association":
				expectedError = models.ErrSubscriptionTaskAssociation
				require.NoError(t, db.Where("key = ? AND value = ?", "task_id", rejected.ID.String()).Delete(&models.CanvasNodeExecutionKV{}).Error)
			}
			otherOrganization, err := models.CreateOrganization("Healthy fleet organization", "")
			require.NoError(t, err)
			healthy := &models.RunnerTask{ID: uuid.New(), OrganizationID: otherOrganization.ID, FleetID: runner.FleetID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
				QueuedAt: time.Now().Add(time.Minute), PayloadCiphertext: []byte("healthy")}
			require.NoError(t, db.Create(healthy).Error)

			reserved, err := runner.ReserveNextTask(db)
			require.NoError(t, err)
			require.NotNil(t, reserved.Task)
			assert.Equal(t, healthy.ID, reserved.Task.ID)
			require.Len(t, reserved.FailedTasks, 1)
			assert.Equal(t, rejected.ID, reserved.FailedTasks[0].ID)
			stored, err := models.FindRunnerTask(db, rejected.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStateFailed, stored.State)
			require.NotNil(t, stored.ErrorMessage)
			assert.Equal(t, expectedError.Error(), *stored.ErrorMessage)
			assert.NotNil(t, stored.FinishedAt)
			assert.Nil(t, stored.RunnerID)
			assert.Empty(t, codexLeases(t, db, connection.ID))
		})
	}
}

func TestRunnerAdmissionCommitsLoneSubscriptionFailureAndReleasesLogin(t *testing.T) {
	db, connection, task, previousRunner := newDisconnectedCodexLogin(t)
	runner := &models.Runner{ID: uuid.New(), FleetID: previousRunner.FleetID, State: models.RunnerStateIdle, RunnerVersion: previousRunner.RunnerVersion}
	require.NoError(t, db.Create(runner).Error)
	require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", task.ID).Update("revoked_at", time.Now()).Error)

	reserved, err := runner.ReserveNextTask(db)
	require.NoError(t, err)
	assert.Nil(t, reserved.Task)
	require.Len(t, reserved.FailedTasks, 1)
	assert.Equal(t, task.ID, reserved.FailedTasks[0].ID)
	require.Len(t, reserved.LoginConnections, 1)
	assert.Equal(t, connection.ID, reserved.LoginConnections[0].ID)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateFailed, stored.State)
	require.NotNil(t, stored.ErrorMessage)
	assert.Equal(t, models.ErrCodexConnectionStale.Error(), *stored.ErrorMessage)
	assert.NotNil(t, stored.FinishedAt)
	assert.Empty(t, codexLeases(t, db, connection.ID))
}

func TestRunnerAdmissionKeepsContendedSubscriptionQueuedAndReleasesLocks(t *testing.T) {
	db, connection, task, runner, order := queuedSubscriptionRunnerTask(t)
	holder := db.Begin()
	require.NoError(t, holder.Error)
	defer holder.Rollback()
	require.NoError(t, order.LockForUpdate(holder))

	scan := db.Begin()
	require.NoError(t, scan.Error)
	defer scan.Rollback()
	reserved, err := runner.ReserveNextTask(scan)
	require.NoError(t, err)
	assert.Nil(t, reserved.Task)
	assert.Empty(t, reserved.FailedTasks)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, stored.State)
	assert.Nil(t, stored.ErrorMessage)
	assert.Nil(t, stored.FinishedAt)
	require.Len(t, codexLeases(t, db, connection.ID), 1)
	require.NoError(t, db.Transaction(func(probe *gorm.DB) error {
		return probe.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).First(&models.RunnerTask{}, "id = ?", task.ID).Error
	}))
	require.NoError(t, scan.Commit().Error)
	require.NoError(t, holder.Rollback().Error)
	reserved, err = runner.ReserveNextTask(db)
	require.NoError(t, err)
	require.NotNil(t, reserved.Task)
	assert.Equal(t, task.ID, reserved.Task.ID)
}

func queuedSubscriptionRunnerTask(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner, *models.FactoryWorkOrder) {
	t.Helper()
	db, connection, task, previousRunner := newReadyCodexConnection(t)
	require.NoError(t, db.Model(task).Updates(map[string]any{"state": models.RunnerTaskStateQueued, "runner_id": nil, "finished_at": nil}).Error)
	runner := &models.Runner{ID: uuid.New(), FleetID: previousRunner.FleetID, State: models.RunnerStateIdle, RunnerVersion: previousRunner.RunnerVersion}
	require.NoError(t, db.Create(runner).Error)
	runID := support.CreateCodexOwnedRun(t, db, connection, task.ID)
	_, err := models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, runID, task.ID, time.Now(), crypto.NewNoOpEncryptor())
	require.NoError(t, err)
	order, err := models.FindSubscriptionWorkOrderForRun(db, connection.OrganizationID, connection.FactoryID, runID)
	require.NoError(t, err)
	return db, connection, task, runner, order
}
