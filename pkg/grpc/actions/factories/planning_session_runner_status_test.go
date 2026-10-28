package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestPlanningSessionReportsRunnerCapacityThroughSessionAndBoardAPIs(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.DB(t.Context())
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	session := openAnalysisSession(t, r, db)
	plan := &billingmodels.BillingPlan{
		Key: "planning_" + uuid.NewString()[:8], Name: "Planning capacity", Category: "paid", Visibility: "private",
		CreditFrequency: "none", CreditExpiration: "never", ParallelTaskLimit: new(1),
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	fleet.Slug = support.RandomName("planning")
	require.NoError(t, fleet.Create(db))
	newRunner := func() *models.Runner {
		runner := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle, RunnerVersion: "test"}
		require.NoError(t, db.Create(runner).Error)
		return runner
	}
	newTask := func() *models.RunnerTask {
		task := &models.RunnerTask{
			ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
			Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
			QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted"),
		}
		require.NoError(t, db.Create(task).Error)
		return task
	}
	occupying := newTask()
	require.NoError(t, occupying.Reserve(db, newRunner().ID))
	queued := newTask()
	event := support.EmitCanvasEventForNode(t, *session.CanvasID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, *session.CanvasID, intakeAnalysisNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"run_id": *session.CanvasRunID, "state": models.CanvasNodeExecutionStateStarted,
	}).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, *session.CanvasID, intakeAnalysisNodeID, execution.ID, "task_id", queued.ID.String()))

	assertStatus := func(state, reason string) {
		t.Helper()
		described, err := DescribePlanningSession(ctx, r.Organization.ID.String(), &pb.DescribePlanningSessionRequest{
			FactoryId: session.FactoryID.String(), SessionId: session.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, state, described.Session.GetRunnerState())
		assert.Equal(t, reason, described.Session.GetQueueReason())
		listed, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{FactoryId: session.FactoryID.String()})
		require.NoError(t, err)
		require.Len(t, listed.Orders, 1)
		assert.Equal(t, state, listed.Orders[0].GetPlanningSession().GetRunnerState())
		assert.Equal(t, reason, listed.Orders[0].GetPlanningSession().GetQueueReason())
	}

	assertStatus(models.RunnerTaskStateQueued, "parallelism_limit")
	plan.ParallelTaskLimit = new(2)
	require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
	assertStatus(models.RunnerTaskStateQueued, "")
	runner := newRunner()
	require.NoError(t, queued.Reserve(db, runner.ID))
	assertStatus(models.RunnerTaskStateReserved, "")
	require.NoError(t, queued.Start(db, runner, "test", time.Now()))
	assertStatus(models.RunnerTaskStateRunning, "")
}
