package factories

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestCancelWorkOrderQueueCancelsOnlyTheExactQueuedRunnerJob(t *testing.T) {
	t.Setenv(models.BillingEnabledEnv, "no")
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, "Queue", "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Verify changes", "", &r.User, nil, nil)
	require.NoError(t, err)
	canvas, nodeID := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Verify", "start")
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, nodeID, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	_, err = factory.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CanvasID: canvas.ID, CanvasRunID: run.ID, WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	child, err := models.CreateCanvasRunInTransaction(db, canvas.ID, nodeID, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(child).Update("parent_run_id", run.ID).Error)
	event := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	fleet := &models.RunnerFleet{ID: uuid.New(), Slug: "queue-jobs", RunnerVersion: "dev", ScopeType: models.RunnerFleetScopeInstallation, Enabled: true}
	require.NoError(t, fleet.Create(db))
	now := time.Now()
	createJob := func(associated bool) *models.RunnerTask {
		job := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
			Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
			PayloadCiphertext: []byte("payload"), QueuedAt: now, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, db.Create(job).Error)
		if associated {
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, nodeID, event.ID, event.ID)
			require.NoError(t, db.Model(execution).Update("run_id", child.ID).Error)
			require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, nodeID, execution.ID, "task_id", job.ID.String()))
		}
		return job
	}
	first, sibling, unassociated := createJob(true), createJob(true), createJob(false)
	queued, err := models.ListOrganizationQueuedRunnerJobs(db, r.Organization.ID)
	require.NoError(t, err)
	require.Len(t, queued, 3)
	for i, job := range queued {
		assert.Equal(t, int64(i+1), job.Position)
	}
	expectedOrder := []string{first.ID.String(), sibling.ID.String(), unassociated.ID.String()}
	sort.Strings(expectedOrder)
	actualOrder := []string{}
	for _, job := range queued {
		actualOrder = append(actualOrder, job.ID.String())
	}
	assert.Equal(t, expectedOrder, actualOrder)
	positions := map[uuid.UUID]int64{}
	for _, job := range queued {
		positions[job.ID] = job.Position
	}
	assert.NotEqual(t, positions[first.ID], positions[sibling.ID])
	assert.NotZero(t, positions[unassociated.ID])

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	request := &pb.CancelWorkOrderQueueRequest{FactoryId: factory.ID.String(), OrderId: order.ID.String(), QueueId: first.ID.String()}
	response, err := CancelWorkOrderQueue(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	require.Len(t, response.Order.QueuedRunnerJobs, 1)
	assert.Equal(t, sibling.ID.String(), response.Order.QueuedRunnerJobs[0].Id)
	canceled, err := models.FindRunnerTask(db, first.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateCanceled, canceled.State)
	retained, err := models.FindRunnerTask(db, sibling.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, retained.State)
	_, err = CancelWorkOrderQueue(ctx, r.Organization.ID.String(), request)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	otherOrder, err := factory.CreateWorkOrder(db, "Other task", "", &r.User, nil, nil)
	require.NoError(t, err)
	request.OrderId = otherOrder.ID.String()
	request.QueueId = sibling.ID.String()
	_, err = CancelWorkOrderQueue(ctx, r.Organization.ID.String(), request)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	runner := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle, RunnerVersion: "dev"}
	require.NoError(t, db.Create(runner).Error)
	require.NoError(t, retained.Reserve(db, runner.ID))
	request.OrderId = order.ID.String()
	_, err = CancelWorkOrderQueue(ctx, r.Organization.ID.String(), request)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	assigned, err := models.FindRunnerTask(db, sibling.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateReserved, assigned.State)
	assert.Nil(t, assigned.CancelRequestedAt)
}
