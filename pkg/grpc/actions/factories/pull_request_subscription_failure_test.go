package factories

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	canvasespb "github.com/superplanehq/superplane/pkg/protos/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestWorkOrderAPIsPreserveNestedQuotaFailureOnPullRequestActivities(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("quota"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review changes", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	canvas, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "review", "start")
	event := support.EmitCanvasEventForNode(t, canvas.ID, entry, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(db, run.ID, "Address review"))
	childEvent := support.EmitCanvasEventForNode(t, canvas.ID, entry, "default", nil)
	child, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, childEvent)
	require.NoError(t, err)
	require.NoError(t, db.Model(child).Update("parent_run_id", run.ID).Error)
	require.NoError(t, pullRequest.LinkRun(db, child.ID, "Review nested app"))
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, entry, childEvent.ID, childEvent.ID)
	require.NoError(t, db.Model(execution).Update("run_id", child.ID).Error)
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	fleet.Slug = support.RandomName("quota")
	require.NoError(t, fleet.Create(db))
	reset := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	result, err := json.Marshal(map[string]any{
		"usage_state": "partial",
		"failure": models.SubscriptionFailure{
			Reason: models.SubscriptionFailureChatGPTQuotaExhausted, SubscriptionOwnerID: r.User.String(),
			LimitID: "weekly", ResetsAt: &reset,
		},
	})
	require.NoError(t, err)
	task := &models.RunnerTask{
		ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateFailed,
		QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted"), Result: datatypes.JSON(result),
	}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, entry, execution.ID, "task_id", task.ID.String()))

	described, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(),
	})
	require.NoError(t, err)
	listed, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	require.Len(t, listed.Orders, 1)
	for _, requests := range [][]*pb.FactoryPullRequest{described.Order.PullRequests, listed.Orders[0].PullRequests} {
		require.Len(t, requests, 1)
		require.Len(t, requests[0].Runs, 2)
		require.Len(t, requests[0].Activities, 2)
		for _, saved := range []*canvasespb.CanvasRunRef{requests[0].Runs[0].Run, requests[0].Activities[0].Run, requests[0].Runs[1].Run, requests[0].Activities[1].Run} {
			assert.Contains(t, []string{run.ID.String(), child.ID.String()}, saved.Id)
			assert.Equal(t, "partial", saved.UsageState)
			require.NotNil(t, saved.SubscriptionFailure)
			assert.Equal(t, models.SubscriptionFailureChatGPTQuotaExhausted, saved.SubscriptionFailure.Reason)
			assert.Equal(t, r.User.String(), saved.SubscriptionFailure.SubscriptionOwnerId)
			assert.Equal(t, "weekly", saved.SubscriptionFailure.LimitId)
			assert.Equal(t, reset, saved.SubscriptionFailure.ResetsAt.AsTime())
		}
	}
}
