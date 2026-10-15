package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestCodexConnectionStaysReservedUntilTheLoginVMIsDeleted(t *testing.T) {
	db, connection, task, runner := newReservedCodexLogin(t)
	now := time.Now()
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+1, []byte("bundle-2"), "ada@example.com", true, now))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateSucceeded, &runner.ID, now))

	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))
	held := reloadCodexConnection(t, db, connection.ID)
	require.NotNil(t, held.ReservationTaskID)
	assert.Equal(t, task.ID, *held.ReservationTaskID)
	assert.Equal(t, models.CodexConnectionStatusFinishing, held.Status)

	require.NoError(t, runner.MarkProviderDeleted(db, now))
	released := reloadCodexConnection(t, db, connection.ID)
	assert.Nil(t, released.ReservationTaskID)
	assert.Equal(t, models.CodexConnectionStatusReady, released.Status)
	assert.NotEqual(t, held.ReservationGeneration, released.ReservationGeneration)
}

func TestCodexConnectionLostRunnerStaysReservedUntilVMDeletion(t *testing.T) {
	db, connection, task, runner := newReadyCodexConnection(t)
	now := time.Now()
	reserved, err := models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, task.ID, now)
	require.NoError(t, err)
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateLost, &runner.ID, now))
	require.NoError(t, models.MarkCodexConnectionUncertain(db, task.ID, now))

	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))
	held := reloadCodexConnection(t, db, reserved.ID)
	require.NotNil(t, held.ReservationTaskID)
	assert.Equal(t, models.CodexConnectionStatusReconnectRequired, held.Status)

	require.NoError(t, runner.MarkProviderDeleted(db, now))
	released := reloadCodexConnection(t, db, reserved.ID)
	assert.Nil(t, released.ReservationTaskID)
	assert.Equal(t, models.CodexConnectionStatusReconnectRequired, released.Status)
}

func TestCodexConnectionRejectsStaleCheckpoint(t *testing.T) {
	db, connection, task, _ := newReservedCodexLogin(t)
	now := time.Now()
	err := connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+2, []byte("bundle"), "ada@example.com", false, now)
	require.ErrorIs(t, err, models.ErrCodexConnectionStale)

	require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+1, []byte("bundle-1"), "ada@example.com", false, now))
	err = connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion, []byte("bundle-old"), "ada@example.com", true, now)
	require.ErrorIs(t, err, models.ErrCodexConnectionStale)
}

func TestCodexConnectionSecondReserveNamesTheActiveTask(t *testing.T) {
	db, connection, task, _ := newReadyCodexConnection(t)
	now := time.Now()
	_, err := models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, task.ID, now)
	require.NoError(t, err)

	_, err = models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, uuid.New(), now)
	var busy *models.CodexConnectionBusyError
	require.ErrorAs(t, err, &busy)
	assert.Equal(t, task.ID, busy.TaskID)
	assert.Contains(t, err.Error(), task.ID.String())
}

func TestCodexConnectionSuccessWithoutFinalCheckpointRequiresReconnect(t *testing.T) {
	db, connection, task, runner := newReadyCodexConnection(t)
	now := time.Now()
	reserved, err := models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, task.ID, now)
	require.NoError(t, err)
	require.NoError(t, reserved.StoreBundle(db, task.ID, reserved.ReservationGeneration, reserved.CredentialVersion+1, []byte("rotated"), "ada@example.com", false, now))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateSucceeded, &runner.ID, now))
	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))

	released := reloadCodexConnection(t, db, connection.ID)
	assert.Nil(t, released.ReservationTaskID)
	assert.Equal(t, models.CodexConnectionStatusReconnectRequired, released.Status)
}

func TestCodexConnectionCancelBeforeCheckpointRestoresPreviousStatus(t *testing.T) {
	db, connection, task, _ := newDisconnectedCodexLogin(t)
	now := time.Now()
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateCanceled, nil, now))
	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))

	released := reloadCodexConnection(t, db, connection.ID)
	assert.Nil(t, released.ReservationTaskID)
	assert.Equal(t, models.CodexConnectionStatusDisconnected, released.Status)
}

func clearProviderDeletion(db *gorm.DB, runner *models.Runner) error {
	runner.ProviderDeletedAt = nil
	return db.Model(runner).Update("provider_deleted_at", nil).Error
}

func newDisconnectedCodexLogin(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner) {
	t.Helper()
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateTerminated,
		RunnerVersion: "0.1.0",
		TerminatedAt:  &now,
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
	connection, err := models.CreateFactoryCodexConnection(db, resource.Organization.ID, factory.ID, resource.User, now)
	require.NoError(t, err)
	require.NoError(t, connection.BeginLogin(db, task.ID, now))
	return db, reloadCodexConnection(t, db, connection.ID), task, runner
}

func newReservedCodexLogin(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner) {
	t.Helper()
	db, connection, task, runner := newReadyCodexConnection(t)
	require.NoError(t, connection.BeginLogin(db, task.ID, time.Now()))
	return db, reloadCodexConnection(t, db, connection.ID), task, runner
}

func newReadyCodexConnection(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner) {
	t.Helper()
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureCodexSubscriptions))

	factory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateTerminated,
		RunnerVersion: "0.1.0",
		TerminatedAt:  &now,
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

	connection, err := models.CreateFactoryCodexConnection(db, resource.Organization.ID, factory.ID, resource.User, now)
	require.NoError(t, err)
	require.NoError(t, connection.BeginLogin(db, task.ID, now))
	require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, 1, []byte("bundle"), "ada@example.com", true, now))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateSucceeded, &runner.ID, now))
	require.NoError(t, runner.MarkProviderDeleted(db, now))
	return db, reloadCodexConnection(t, db, connection.ID), task, reloadRunner(t, db, runner.ID)
}

func markRunnerTask(db *gorm.DB, task *models.RunnerTask, state string, runnerID *uuid.UUID, now time.Time) error {
	return db.Model(task).Updates(map[string]any{
		"state":       state,
		"runner_id":   runnerID,
		"finished_at": now,
		"updated_at":  now,
	}).Error
}

func reloadCodexConnection(t *testing.T, db *gorm.DB, id uuid.UUID) *models.FactoryCodexConnection {
	t.Helper()
	var connection models.FactoryCodexConnection
	require.NoError(t, db.First(&connection, "id = ?", id).Error)
	return &connection
}

func reloadRunner(t *testing.T, db *gorm.DB, id uuid.UUID) *models.Runner {
	t.Helper()
	var runner models.Runner
	require.NoError(t, db.First(&runner, "id = ?", id).Error)
	return &runner
}
