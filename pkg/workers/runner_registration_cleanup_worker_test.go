package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerCleanupRefreshesQueuedPlanningAfterCapacityRelease(t *testing.T) {
	for _, state := range []string{models.RunnerStatePending, models.RunnerStateIdle, models.RunnerStateBusy} {
		t.Run(state, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			fleet := models.DefaultInstallationRunnerFleets("test")[0]
			fleet.Slug = support.RandomName("cleanup")
			require.NoError(t, fleet.Create(db))
			lastSeen := time.Now().Add(-runnerConnectionLossTimeout - time.Minute)
			runner := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle, RunnerVersion: "test", LastSeenAt: &lastSeen}
			if state == models.RunnerStatePending {
				runner.State = state
			}
			require.NoError(t, db.Create(runner).Error)
			occupying := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("encrypted"), QueuedAt: time.Now()}
			require.NoError(t, db.Create(occupying).Error)
			require.NoError(t, occupying.Reserve(db, runner.ID))
			if state == models.RunnerStateBusy {
				require.NoError(t, occupying.Start(db, runner, "test", time.Now()))
			}
			if state == models.RunnerStatePending {
				require.NoError(t, db.Create(&models.RunnerRegistration{JTI: uuid.New(), RunnerID: runner.ID, ExpiresAt: time.Now().Add(-time.Minute)}).Error)
			}
			factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			order, err := factoryModel.CreateWorkOrder(db, "Queued planning", "", &r.User, nil, nil)
			require.NoError(t, err)
			canvas, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "Backlog", "start")
			run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, entry, models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			_, err = factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
				Repository: "acme/payments", CanvasID: canvas.ID, CanvasRunID: run.ID, WorkOrderID: order.ID,
			})
			require.NoError(t, err)
			queued := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("encrypted"), QueuedAt: time.Now()}
			require.NoError(t, db.Create(queued).Error)
			event := support.EmitCanvasEventForNode(t, canvas.ID, entry, "default", nil)
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, entry, event.ID, event.ID)
			require.NoError(t, db.Model(execution).Update("run_id", run.ID).Error)
			require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, entry, execution.ID, "task_id", queued.ID.String()))
			var releasedStates []string
			restore := messages.SetFactoryWorkOrderUpdatedPublisherForTest(func(factoryID, orderID, reason string) error {
				if factoryID != factoryModel.ID.String() || orderID != "" || reason != "runner_capacity" {
					return nil
				}
				stored, err := models.FindRunnerTask(database.DB(t.Context()), occupying.ID)
				if err != nil {
					return err
				}
				releasedStates = append(releasedStates, stored.State)
				return nil
			})
			defer restore()
			worker := NewRunnerCleanupWorker(time.Minute)
			require.NoError(t, worker.Process())
			expected := models.RunnerTaskStateLost
			if state == models.RunnerStatePending {
				expected = models.RunnerTaskStateQueued
			}
			assert.Equal(t, []string{expected}, releasedStates)
			require.NoError(t, worker.Process())
			assert.Len(t, releasedStates, 1)
		})
	}
}
