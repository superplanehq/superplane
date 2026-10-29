package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerTaskSchedulerRecoversPendingWorkAfterSlotRelease(t *testing.T) {
	resource := support.Setup(t)
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.DB(t.Context())
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	require.NoError(t, fleet.Create(db))
	plan := &billingmodels.BillingPlan{
		Key: "scheduler_" + uuid.NewString()[:8], Name: "Scheduler capacity", Category: "paid", Visibility: "private",
		CreditFrequency: "none", CreditExpiration: "never", ParallelTaskLimit: new(1),
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
	_, err := billingmodels.SetAdminOrganizationPlan(db, resource.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	now := time.Now()
	first := models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, ReadyAt: &now,
		QueuedAt: now, PayloadCiphertext: []byte("encrypted")}
	require.NoError(t, db.Create(&first).Error)
	next := models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStatePending,
		QueuedAt: now, PayloadCiphertext: []byte("encrypted")}
	require.NoError(t, db.Create(&next).Error)
	worker := NewRunnerTaskScheduler(time.Second)
	require.NoError(t, worker.Process(t.Context()))
	stored, err := models.FindRunnerTask(db, next.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStatePending, stored.State)
	require.NoError(t, first.RequestCancel(db, time.Now()))
	// No notification is needed to recover work after a released slot or restart.
	worker = NewRunnerTaskScheduler(time.Second)
	require.NoError(t, worker.Process(t.Context()))
	stored, err = models.FindRunnerTask(db, next.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, stored.State)
	require.NotNil(t, stored.ReadyAt)
	assert.False(t, stored.ReadyAt.Before(now))
}
