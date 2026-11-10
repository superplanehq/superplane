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

func TestRunnerTaskSchedulerRecoversSlotsAndPreservesAssignedTasks(t *testing.T) {
	for _, scenario := range []string{"cancellation", "disabled fleet", "deleted fleet", "returned reservation", "reserved task", "running task", "disabled without pending work"} {
		t.Run(scenario, func(t *testing.T) {
			resource := support.Setup(t)
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			db := database.DB(t.Context())
			fleets := models.DefaultInstallationRunnerFleets("test")
			firstFleet, nextFleet := &fleets[0], &fleets[1]
			require.NoError(t, firstFleet.Create(db))
			require.NoError(t, nextFleet.Create(db))
			plan := &billingmodels.BillingPlan{
				Key: "scheduler_" + uuid.NewString()[:8], Name: "Scheduler capacity", Category: "paid", Visibility: "private",
				CreditFrequency: "none", CreditExpiration: "never", ParallelTaskLimit: new(1),
			}
			require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
			_, err := billingmodels.SetAdminOrganizationPlan(db, resource.Organization.ID, plan.Key, nil)
			require.NoError(t, err)
			now := time.Now()
			first := models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: firstFleet.ID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, ReadyAt: &now,
				QueuedAt: now, PayloadCiphertext: []byte("encrypted")}
			require.NoError(t, db.Create(&first).Error)
			next := models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: nextFleet.ID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStatePending,
				QueuedAt: now.Add(time.Millisecond), PayloadCiphertext: []byte("encrypted")}
			require.NoError(t, db.Create(&next).Error)
			worker := NewRunnerTaskScheduler(time.Second)
			require.NoError(t, worker.Process(t.Context()))
			stored, err := models.FindRunnerTask(db, next.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStatePending, stored.State)

			switch scenario {
			case "cancellation":
				require.NoError(t, first.RequestCancel(db, time.Now()))
			case "returned reservation", "reserved task", "running task":
				runner := &models.Runner{ID: uuid.New(), FleetID: firstFleet.ID, State: models.RunnerStateIdle, RunnerVersion: "test"}
				require.NoError(t, db.Create(runner).Error)
				require.NoError(t, first.Reserve(db, runner.ID))
				if scenario == "running task" {
					require.NoError(t, first.Start(db, runner, "test-store", time.Now()))
				}
				require.NoError(t, firstFleet.SetEnabled(db, false))
				if scenario == "returned reservation" {
					require.NoError(t, runner.Terminate(db, models.RunnerTerminationProvisioningFailed))
				}
			case "deleted fleet":
				require.NoError(t, db.Delete(firstFleet).Error)
			default:
				require.NoError(t, firstFleet.SetEnabled(db, false))
			}
			if scenario == "disabled without pending work" {
				require.NoError(t, next.RequestCancel(db, time.Now()))
			}
			// A new worker must recover the durable state without a notification.
			worker = NewRunnerTaskScheduler(time.Second)
			require.NoError(t, worker.Process(t.Context()))
			stored, err = models.FindRunnerTask(db, next.ID)
			require.NoError(t, err)
			if scenario == "reserved task" || scenario == "running task" {
				assert.Equal(t, models.RunnerTaskStatePending, stored.State)
				assigned, err := models.FindRunnerTask(db, first.ID)
				require.NoError(t, err)
				assert.Equal(t, first.State, assigned.State)
				assert.Equal(t, first.RunnerID, assigned.RunnerID)
				return
			}
			if scenario == "disabled without pending work" {
				assert.Equal(t, models.RunnerTaskStateCanceled, stored.State)
			} else {
				assert.Equal(t, models.RunnerTaskStateQueued, stored.State)
				require.NotNil(t, stored.ReadyAt)
				assert.False(t, stored.ReadyAt.Before(now))
			}
			if scenario == "cancellation" {
				return
			}
			deferred, err := models.FindRunnerTask(db, first.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStatePending, deferred.State)
			assert.Nil(t, deferred.ReadyAt)
			assert.Equal(t, first.QueuedAt.UnixMicro(), deferred.QueuedAt.UnixMicro())
			if scenario == "disabled fleet" {
				require.NoError(t, firstFleet.SetEnabled(db, true))
				require.NoError(t, stored.RequestCancel(db, time.Now()))
				require.NoError(t, worker.Process(t.Context()))
				restored, err := models.FindRunnerTask(db, first.ID)
				require.NoError(t, err)
				assert.Equal(t, models.RunnerTaskStateQueued, restored.State)
				assert.NotNil(t, restored.ReadyAt)
			}
		})
	}
}
