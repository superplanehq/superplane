package factories

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	canvasespb "github.com/superplanehq/superplane/pkg/protos/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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
	require.NoError(t, db.Model(queued).Update("state", models.RunnerTaskStatePending).Error)
	event := support.EmitCanvasEventForNode(t, *session.CanvasID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, *session.CanvasID, intakeAnalysisNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"run_id": *session.CanvasRunID, "state": models.CanvasNodeExecutionStateStarted,
	}).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, *session.CanvasID, intakeAnalysisNodeID, execution.ID, "task_id", queued.ID.String()))

	assertStatus := func(state string, blocked bool) {
		t.Helper()
		described, err := DescribePlanningSession(ctx, r.Organization.ID.String(), &pb.DescribePlanningSessionRequest{
			FactoryId: session.FactoryID.String(), SessionId: session.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, state, described.Session.GetRunnerState())
		assert.Equal(t, blocked, described.Session.GetBlockedByParallelismLimit())
		listed, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{FactoryId: session.FactoryID.String()})
		require.NoError(t, err)
		require.Len(t, listed.Orders, 1)
		assert.Equal(t, state, listed.Orders[0].GetPlanningSession().GetRunnerState())
		assert.Equal(t, blocked, listed.Orders[0].GetPlanningSession().GetBlockedByParallelismLimit())
	}

	assertStatus(models.RunnerTaskStatePending, true)
	plan.ParallelTaskLimit = new(2)
	require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
	assertStatus(models.RunnerTaskStatePending, false)
	_, err = models.ScheduleOrganizationRunnerTasks(db, r.Organization.ID, 100)
	require.NoError(t, err)
	assertStatus(models.RunnerTaskStateQueued, false)
	runner := newRunner()
	require.NoError(t, queued.Reserve(db, runner.ID))
	assertStatus(models.RunnerTaskStateReserved, false)
	require.NoError(t, queued.Start(db, runner, "test", time.Now()))
	assertStatus(models.RunnerTaskStateRunning, false)
}

func TestPlanningQuotaFailureSurvivesReloadAndClearsForNewTask(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	session := openAnalysisSession(t, r, db)
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	fleet.Slug = support.RandomName("quota")
	require.NoError(t, fleet.Create(db))
	result, err := json.Marshal(map[string]any{"usage_state": "partial", "failure": models.SubscriptionFailure{Reason: models.SubscriptionFailureChatGPTQuotaExhausted, SubscriptionOwnerID: r.User.String()}})
	require.NoError(t, err)
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateFailed, QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted"), Result: datatypes.JSON(result)}
	require.NoError(t, db.Create(task).Error)
	event := support.EmitCanvasEventForNode(t, *session.CanvasID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, *session.CanvasID, intakeAnalysisNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Update("run_id", *session.CanvasRunID).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, *session.CanvasID, intakeAnalysisNodeID, execution.ID, "task_id", task.ID.String()))
	assertFailure := func(want bool) {
		t.Helper()
		response, err := DescribePlanningSession(ctx, r.Organization.ID.String(), &pb.DescribePlanningSessionRequest{FactoryId: session.FactoryID.String(), SessionId: session.ID.String()})
		require.NoError(t, err)
		listed, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{FactoryId: session.FactoryID.String()})
		require.NoError(t, err)
		require.Len(t, listed.Orders, 1)
		if want {
			assert.Equal(t, "partial", response.Session.GetUsageState())
			assert.Equal(t, "partial", listed.Orders[0].GetPlanningSession().GetUsageState())
		}
		for _, failure := range []*canvasespb.SubscriptionFailure{response.Session.GetSubscriptionFailure(), listed.Orders[0].GetPlanningSession().GetSubscriptionFailure()} {
			if !want {
				assert.Nil(t, failure)
				continue
			}
			require.NotNil(t, failure)
			assert.Equal(t, "chatgpt_quota_exhausted", failure.GetReason())
			assert.Equal(t, r.User.String(), failure.GetSubscriptionOwnerId())
		}
	}
	assertFailure(true)
	next := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted")}
	require.NoError(t, db.Create(next).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, *session.CanvasID, intakeAnalysisNodeID, execution.ID, "task_id", next.ID.String()))
	assertFailure(false)
	reloaded, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	require.NotNil(t, models.SubscriptionFailureFromResult(reloaded.Result), "historical failure must remain saved")
}
