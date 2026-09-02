package canvases

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/approval"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

const (
	approvalTriggerNodeID = "trigger-1"
	approvalNodeID        = "approval-1"
)

func Test__InvokeNodeExecutionHook__ConcurrentApprovalsAreNotLost(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	stats, err := database.PoolStats()
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.MaxOpenConnections, 4,
		"this test holds three connections and polls on a fourth; set DB_POOL_SIZE >= 4")

	secondApprover := support.CreateUser(t, r, r.Organization.ID)
	canvas, execution := setupApprovalExecution(t, r, r.UserModel, secondApprover)

	//
	// Hold the execution row from a separate transaction, so both hook calls
	// are parked at the same observable point before either of them writes.
	// On the unfixed handler, a plain SELECT does not wait on this lock, so
	// both calls still read the row and reach their metadata write with the
	// same [pending, pending] snapshot - which is what two people approving
	// at the same time does. The fixed handler parks in its SELECT ... FOR NO
	// KEY UPDATE instead, and the second call reads the row after the first
	// one has committed.
	//
	barrier := database.Conn().Begin()
	require.NoError(t, barrier.Error)
	defer barrier.Rollback()

	var lockedExecution models.CanvasNodeExecution
	require.NoError(t, barrier.
		Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).
		Where("id = ?", execution.ID).
		First(&lockedExecution).
		Error)

	results := make(chan error, 2)
	approve := func(userID string, index int) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), userID)
		_, err := InvokeNodeExecutionHook(
			ctx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			database.DB(ctx),
			canvas,
			execution.ID,
			"approve",
			map[string]any{"index": float64(index)},
		)

		results <- err
	}

	go approve(r.User.String(), 0)
	waitForBlockedExecutionWriters(t, 1, results)

	go approve(secondApprover.ID.String(), 1)
	waitForBlockedExecutionWriters(t, 2, results)

	require.NoError(t, barrier.Commit().Error)

	for i := 0; i < 2; i++ {
		select {
		case err := <-results:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatalf("approval %d did not return within 10s after the row lock was released", i+1)
		}
	}

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)

	metadata := approvalMetadataOf(t, updatedExecution)
	require.Len(t, metadata.Records, 2)
	assert.Equal(t, approval.StateApproved, metadata.Records[0].State, "the first approver's decision was overwritten")
	assert.Equal(t, approval.StateApproved, metadata.Records[1].State, "the second approver's decision was overwritten")
	assert.Equal(t, approval.StateApproved, metadata.Result, "the node is still waiting for an approval that was already given")

	events, err := models.ListCanvasEvents(database.DB(t.Context()), canvas.ID, approvalNodeID, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1, "a fully approved node emits exactly one event")
	assert.Equal(t, approval.ChannelApproved, events[0].Channel)

	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State, "the node is still waiting although everyone approved")
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
}

func Test__InvokeNodeExecutionHook__RejectsFinishedExecution(t *testing.T) {
	//
	// The component's own guard reads the in-memory snapshot the handler
	// loaded before the hook ran, and the approval component does not check
	// the state at all - so the handler has to reject the call itself.
	//
	for _, state := range []string{
		models.CanvasNodeExecutionStateFinished,
		models.CanvasNodeExecutionStateCancelling,
	} {
		t.Run(state, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()

			secondApprover := support.CreateUser(t, r, r.Organization.ID)
			canvas, execution := setupApprovalExecution(t, r, r.UserModel, secondApprover)
			require.NoError(t, database.Conn().Model(execution).Update("state", state).Error)

			before, err := models.FindNodeExecution(canvas.ID, execution.ID)
			require.NoError(t, err)
			metadataBefore, err := json.Marshal(before.Metadata.Data())
			require.NoError(t, err)

			ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
			_, err = InvokeNodeExecutionHook(
				ctx,
				r.AuthService,
				r.Encryptor,
				r.Registry,
				database.DB(ctx),
				canvas,
				execution.ID,
				"approve",
				map[string]any{"index": float64(0)},
			)

			assert.Error(t, err, "approving a %s execution must be refused", state)
			assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

			after, err := models.FindNodeExecution(canvas.ID, execution.ID)
			require.NoError(t, err)
			metadataAfter, err := json.Marshal(after.Metadata.Data())
			require.NoError(t, err)
			assert.JSONEq(t, string(metadataBefore), string(metadataAfter), "a %s execution must not record an approval", state)
		})
	}
}

func Test__InvokeNodeExecutionHook__RollsBackMetadataWhenHookFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, execution := setupApprovalExecution(t, r, r.UserModel)

	//
	// One approver completes the node, so the hook emits right after its
	// metadata write. A payload limit of one byte makes that emit fail.
	//
	t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "1")

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	_, err := InvokeNodeExecutionHook(
		ctx,
		r.AuthService,
		r.Encryptor,
		r.Registry,
		database.DB(ctx),
		canvas,
		execution.ID,
		"approve",
		map[string]any{"index": float64(0)},
	)
	require.Error(t, err)

	after, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)

	metadata := approvalMetadataOf(t, after)
	require.Len(t, metadata.Records, 1)
	assert.Equal(t, approval.StatePending, metadata.Records[0].State, "a failed hook must not leave a recorded approval behind")
	assert.Equal(t, approval.StatePending, metadata.Result)
}

// setupApprovalExecution builds a canvas whose approval node needs every
// given user, plus an execution parked on that node with one pending record
// per approver - the state the node is in while people see an Approve button.
func setupApprovalExecution(t *testing.T, r *support.ResourceRegistry, approvers ...*models.User) (*models.Canvas, *models.CanvasNodeExecution) {
	t.Helper()

	items := make([]any, 0, len(approvers))
	records := make([]approval.Record, 0, len(approvers))
	for index, approver := range approvers {
		items = append(items, map[string]any{"type": approval.ItemTypeUser, "user": approver.ID.String()})
		records = append(records, approval.Record{Index: index, Type: approval.ItemTypeUser, State: approval.StatePending, User: approverRecordUser(approver)})
	}

	configuration := map[string]any{"items": items}

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: approvalTriggerNodeID,
				Name:   approvalTriggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID:        approvalNodeID,
				Name:          approvalNodeID,
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "approval"}}),
				Configuration: datatypes.NewJSONType(configuration),
			},
		},
		[]models.Edge{{SourceID: approvalTriggerNodeID, TargetID: approvalNodeID, Channel: "default"}},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, approvalTriggerNodeID, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, approvalNodeID, rootEvent.ID, rootEvent.ID, configuration)

	//
	// Write the records through the same metadata context the component uses,
	// so the fixture cannot drift from what production leaves behind.
	//
	metadata := approval.Metadata{
		Result:  approval.StatePending,
		Records: records,
	}

	require.NoError(t, contexts.NewExecutionMetadataContext(database.Conn(), execution).Set(&metadata))
	require.NoError(t, execution.StartInTransaction(database.Conn()))

	return canvas, execution
}

// waitForBlockedExecutionWriters waits until the expected number of backends
// are waiting on a lock on workflow_node_executions. The barrier is tied to a
// state the database reports, so no test step depends on timing.
func waitForBlockedExecutionWriters(t *testing.T, expected int, results <-chan error) {
	t.Helper()

	const (
		interval = 20 * time.Millisecond
		timeout  = 10 * time.Second
	)

	blocked := 0
	deadline := time.Now().Add(timeout)
	for {
		select {
		case err := <-results:
			t.Fatalf("an approval returned (err=%v) before it reached the locked row: the barrier did not hold and the test proves nothing", err)
		default:
		}

		require.NoError(t, database.Conn().Raw(`
			SELECT count(*)
			FROM pg_stat_activity
			WHERE datname = current_database()
			AND pid <> pg_backend_pid()
			AND wait_event_type = 'Lock'
			AND query ILIKE '%workflow_node_executions%'
		`).Scan(&blocked).Error)

		if blocked >= expected {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("waited %s for %d approval(s) to block on the execution row, saw %d", timeout, expected, blocked)
		}

		time.Sleep(interval)
	}
}

func approverRecordUser(user *models.User) *core.User {
	return &core.User{ID: user.ID.String(), Name: user.Name, Email: user.GetEmail()}
}

func approvalMetadataOf(t *testing.T, execution *models.CanvasNodeExecution) approval.Metadata {
	t.Helper()

	raw, err := json.Marshal(execution.Metadata.Data())
	require.NoError(t, err)

	var metadata approval.Metadata
	require.NoError(t, json.Unmarshal(raw, &metadata))
	return metadata
}
