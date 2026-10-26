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
	"gorm.io/gorm"
)

func TestRunnerPreparationFencesCompetingAttemptsAndKeepsAcceptedPayload(t *testing.T) {
	db, runner, task, connectionID, now := preparationFixture(t)
	first, err := models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now)
	require.NoError(t, err)
	_, err = models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now.Add(time.Second))
	require.ErrorIs(t, err, models.ErrRunnerPreparationUnavailable)
	require.NoError(t, first.Retry(db, now.Add(2*time.Second), now.Add(time.Second)))
	second, err := models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now.Add(3*time.Second))
	require.NoError(t, err)
	assert.Equal(t, first.Deadline, second.Deadline)
	assert.NotEqual(t, first.AttemptID, second.AttemptID)
	require.ErrorIs(t, first.Accept(db, runner.ID, connectionID, []byte("stale"), now.Add(time.Hour), now.Add(4*time.Second)), models.ErrRunnerPreparationUnavailable)
	require.ErrorIs(t, task.FailPreparation(db, "stale failure", first, now.Add(4*time.Second)), models.ErrRunnerPreparationUnavailable)
	cleanup := models.RunnerCredentialCleanup{ID: uuid.New(), TaskID: task.ID, AttemptID: *second.AttemptID, Provider: "openrouter", ProviderResourceID: uuid.New(), Name: uuid.NewString(), ExpiresAt: now.Add(time.Hour), NextRetryAt: now, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, cleanup.Create(db))
	require.NoError(t, second.Accept(db, runner.ID, connectionID, []byte("accepted"), now.Add(20*time.Minute), now.Add(4*time.Second)))
	reused, err := models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, second.AttemptID, reused.AttemptID)
	assert.Equal(t, second.AttemptCount, reused.AttemptCount)
	require.NotNil(t, reused.PreparedAt)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("accepted"), stored.PayloadCiphertext)
	require.NoError(t, db.First(&cleanup, "id = ?", cleanup.ID).Error)
	assert.NotNil(t, cleanup.AcceptedAt, "payload and provider-key ownership commit together")
	require.ErrorIs(t, task.FailPreparation(db, "late stale failure", first, now.Add(time.Minute)), models.ErrRunnerPreparationUnavailable)
	oldVersion := *second
	require.NoError(t, db.Model(&models.RunnerTaskPreparation{}).Where("task_id = ?", task.ID).Update("prepared_valid_until", now.Add(30*time.Second)).Error)
	third, err := models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now.Add(2*time.Minute))
	require.NoError(t, err)
	require.NoError(t, third.Accept(db, runner.ID, connectionID, []byte("replacement"), now.Add(time.Hour), now.Add(2*time.Minute)))
	require.ErrorIs(t, task.BeginDispatch(db, runner.ID, connectionID, &oldVersion, now.Add(2*time.Minute)), models.ErrRunnerPreparationUnavailable)
	require.NoError(t, task.BeginDispatch(db, runner.ID, connectionID, third, now.Add(2*time.Minute)))
	require.NoError(t, task.RequestCancel(db, now.Add(2*time.Minute)))
	stored, err = models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateReserved, stored.State, "dispatch uncertainty retains assigned capacity until acknowledgment or runner loss")
}

func TestRunnerPreparationExpiresAfterRequeueWithoutStartingExecution(t *testing.T) {
	db, runner, task, connectionID, now := preparationFixture(t)
	preparation, err := models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now)
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockRunner(tx, runner.ID); err != nil {
			return err
		}
		return task.ReleaseUndelivered(tx, runner.ID, now.Add(time.Minute))
	}))
	queued, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	require.Equal(t, models.RunnerTaskStateQueued, queued.State)
	assert.Nil(t, queued.RunnerID)
	expired, err := models.ListExpiredRunnerPreparations(db, preparation.Deadline.Add(time.Second))
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.NoError(t, expired[0].FailPreparation(db, "Preparation exceeded five minutes.", nil, preparation.Deadline.Add(time.Second)))
	failed, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateFailed, failed.State)
	assert.Nil(t, failed.StartedAt)
	assert.NotNil(t, failed.FinishedAt)
}

func TestRunnerPreparationCancellationRejectsMintedCredentials(t *testing.T) {
	db, runner, task, connectionID, now := preparationFixture(t)
	preparation, err := models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now)
	require.NoError(t, err)
	cleanup := models.RunnerCredentialCleanup{ID: uuid.New(), TaskID: task.ID, AttemptID: *preparation.AttemptID, Provider: "openrouter", ProviderResourceID: uuid.New(), Name: uuid.NewString(), KeyHash: "minted-before-cancel", ExpiresAt: now.Add(time.Hour), NextRetryAt: now, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, cleanup.Create(db))
	require.NoError(t, task.RequestCancel(db, now.Add(time.Second)))
	require.ErrorIs(t, preparation.Accept(db, runner.ID, connectionID, []byte("rejected"), now.Add(time.Hour), now.Add(2*time.Second)), models.ErrRunnerPreparationUnavailable)
	pending, err := models.ListRunnerCredentialCleanup(db, now.Add(3*time.Second))
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, cleanup.ID, pending[0].ID)
	assert.Nil(t, pending[0].AcceptedAt)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateCanceled, stored.State)
	assert.Nil(t, stored.StartedAt)
}

func preparationFixture(t *testing.T) (*gorm.DB, *models.Runner, *models.RunnerTask, uuid.UUID, time.Time) {
	t.Helper()
	resource := support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	now := time.Now().UTC().Truncate(time.Microsecond)
	connectionID := uuid.New()
	runner := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle, RunnerVersion: "0.1.0", CurrentConnectionID: &connectionID, RegisteredAt: &now, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(runner).Error)
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("saved"), QueuedAt: now.Add(-time.Hour), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))
	return db, runner, task, connectionID, now
}
