package runnerapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestPreparationExpiryReleasesJobsWhenNotificationQueueIsFull(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	t.Setenv(models.BillingEnabledEnv, "no")
	db := database.DB(t.Context())
	now := time.Now()
	runner := &models.Runner{ID: uuid.New(), FleetID: fixture.task.FleetID, State: models.RunnerStateIdle,
		RunnerVersion: "dev", CreatedAt: now, UpdatedAt: now}
	connectionID := uuid.New()
	runner.CurrentConnectionID = &connectionID
	require.NoError(t, db.Create(runner).Error)
	second := &models.RunnerTask{ID: uuid.New(), OrganizationID: fixture.task.OrganizationID, FleetID: fixture.task.FleetID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte(`{}`), QueuedAt: now, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(second).Error)
	require.NoError(t, second.Reserve(db, runner.ID))
	_, err := models.ClaimRunnerTaskPreparation(db, second.ID, runner.ID, connectionID, now)
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.RunnerTaskPreparation{}).
		Where("task_id IN ?", []uuid.UUID{fixture.task.ID, second.ID}).Update("deadline", now.Add(-time.Second)).Error)

	// No consumer can receive a notification. Durable failure and capacity
	// release must still complete for every expired job in this sweep.
	fixture.server.sweepPreparations(t.Context(), make(chan *models.RunnerTask))
	for _, id := range []uuid.UUID{fixture.task.ID, second.ID} {
		job, err := models.FindRunnerTask(db, id)
		require.NoError(t, err)
		require.Equal(t, models.RunnerTaskStateFailed, job.State)
		require.Nil(t, job.StartedAt)
		require.Nil(t, job.DispatchStartedAt)
	}
	active, err := models.CountActiveOrganizationRunnerJobs(db, fixture.task.OrganizationID)
	require.NoError(t, err)
	require.Zero(t, active)
}
