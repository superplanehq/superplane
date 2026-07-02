package workers

import (
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newTestNodeExecutor(t *testing.T, r *support.ResourceRegistry) *NodeExecutor {
	t.Helper()
	return NewNodeExecutor(
		r.Encryptor,
		r.Registry,
		r.GitProvider,
		support.NewOIDCProvider(),
		"http://localhost",
		"http://localhost",
		"",
		r.AuthService,
	)
}

func Test__NodeExecutor_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple canvas with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a pending execution for the component node.
	//
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)

	//
	// Have two workers call LockAndProcessNodeExecution concurrently on the same execution.
	// LockAndProcessNodeExecution uses a transaction with SKIP LOCKED, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the execution concurrently.
	//
	go func() {
		executor1 := newTestNodeExecutor(t, r)
		results <- executor1.LockAndProcessNodeExecution(execution.ID)
	}()

	go func() {
		executor2 := newTestNodeExecutor(t, r)
		results <- executor2.LockAndProcessNodeExecution(execution.ID)
	}()

	// Collect results - one should succeed (return nil) and one should get ErrRecordLocked
	// because LockAndProcessNodeExecution returns ErrRecordLocked when it can't acquire the lock
	result1 := <-results
	result2 := <-results

	successCount, lockedCount := countConcurrentExecutionResults(t, []error{result1, result2})
	assert.Equal(t, 1, successCount, "Exactly one worker should succeed")
	assert.Equal(t, 1, lockedCount, "Exactly one worker should get ErrRecordLocked")

	//
	// Verify the execution was started and finished (since noop completes immediately).
	// If both workers processed it, we would see inconsistent state or errors.
	//
	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
}

func Test__NodeExecutor_DoesNotProcessExecutionForSoftDeletedOrganization(t *testing.T) {
	r := support.Setup(t)

	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}})},
			{NodeID: componentNode, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	executions, err := models.ListPendingNodeExecutions()
	require.NoError(t, err)
	for _, pending := range executions {
		assert.NotEqual(t, execution.ID, pending.ID)
	}

	executor := newTestNodeExecutor(t, r)
	err = executor.LockAndProcessNodeExecution(execution.ID)
	assert.ErrorIs(t, err, ErrRecordLocked)

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStatePending, updatedExecution.State)
	assert.Empty(t, updatedExecution.Result)
}

func Test__NodeExecutor_ComponentNodeWithoutStateChange(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple canvas with a trigger and an approval component node.
	// The approval component does NOT change state on Execute() - it just sets metadata.
	//
	triggerNode := "trigger-1"
	approvalNode := "approval-1"
	approvalConfiguration := map[string]any{
		"items": []any{
			map[string]any{
				"type": "user",
				"user": r.User.String(),
			},
		},
	}

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID:        approvalNode,
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "approval"}}),
				Configuration: datatypes.NewJSONType(approvalConfiguration),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: approvalNode, Channel: "default"},
		},
	)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)

	log.Printf("nodes: %v", nodes)

	//
	// Create a root event and a pending execution for the approval node.
	//
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, approvalNode, rootEvent.ID, rootEvent.ID, approvalConfiguration)

	//
	// Process the execution and verify the execution is started but NOT finished.
	// The approval component doesn't call Pass() in Execute(), so it should remain in started state.
	//
	executor := newTestNodeExecutor(t, r)
	err = executor.LockAndProcessNodeExecution(execution.ID)
	require.NoError(t, err)

	// Verify execution moved to started state but not finished,
	// and metadata is updated.
	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateStarted, updatedExecution.State)
	assert.Equal(t, "", updatedExecution.Result)
	assert.Equal(t, map[string]any{
		"result": "pending",
		"records": []any{
			map[string]any{
				"index": float64(0),
				"type":  "user",
				"state": "pending",
				"user": map[string]any{
					"id":    r.User.String(),
					"name":  r.UserModel.Name,
					"email": r.UserModel.GetEmail(),
				},
			},
		},
	}, updatedExecution.Metadata.Data())
}

func Test__NodeExecutor_ComponentNodeWithStateChange(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple canvas with a trigger and a noop component node.
	// The noop component DOES change state on Execute() - it calls Pass() immediately.
	//
	triggerNode := "trigger-1"
	noopNode := "noop-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: noopNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: noopNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a pending execution for the noop node.
	//
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, noopNode, rootEvent.ID, rootEvent.ID)

	//
	// Process the execution and verify the execution is both started AND finished.
	// The noop component calls Pass() in Execute(), which should finish the execution.
	//
	executor := newTestNodeExecutor(t, r)
	err := executor.LockAndProcessNodeExecution(execution.ID)
	require.NoError(t, err)

	// Verify execution moved to finished state with passed result
	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
}

func countConcurrentExecutionResults(t *testing.T, results []error) (successCount int, lockedCount int) {
	for i, result := range results {
		switch result {
		case nil:
			successCount++
		case ErrRecordLocked:
			lockedCount++
		default:
			t.Fatalf("Unexpected error from worker %d: %v", i+1, result)
		}
	}
	return successCount, lockedCount
}

func TestClassifyProcessError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: executorReasonNone},
		{name: "locked", err: ErrRecordLocked, want: executorReasonLocked},
		{name: "not found", err: gorm.ErrRecordNotFound, want: executorReasonNotFound},
		{name: "deadlock", err: errors.New("ERROR: deadlock detected (SQLSTATE 40P01)"), want: executorReasonDeadlock},
		{name: "pg deadlock code", err: &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}, want: executorReasonDeadlock},
		{name: "wrapped pg deadlock", err: fmt.Errorf("update execution: %w", &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}), want: executorReasonDeadlock},
		{name: "internal", err: errors.New("something else"), want: executorReasonInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyProcessError(tt.err); got != tt.want {
				t.Fatalf("classifyProcessError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyExecutionFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		execution models.CanvasNodeExecution
		want      string
	}{
		{
			name: "passed execution",
			execution: models.CanvasNodeExecution{
				Result: models.CanvasNodeExecutionResultPassed,
			},
			want: executorReasonNone,
		},
		{
			name: "component error",
			execution: models.CanvasNodeExecution{
				Result:        models.CanvasNodeExecutionResultFailed,
				ResultReason:  models.CanvasNodeExecutionResultReasonError,
				ResultMessage: "request failed",
			},
			want: models.CanvasNodeExecutionResultReasonError,
		},
		{
			name: "deadlock message",
			execution: models.CanvasNodeExecution{
				Result:        models.CanvasNodeExecutionResultFailed,
				ResultReason:  models.CanvasNodeExecutionResultReasonError,
				ResultMessage: "ERROR: deadlock detected (SQLSTATE 40P01)",
			},
			want: executorReasonDeadlock,
		},
		{
			name: "custom reason",
			execution: models.CanvasNodeExecution{
				Result:       models.CanvasNodeExecutionResultFailed,
				ResultReason: "timeout",
			},
			want: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyExecutionFailure(&tt.execution); got != tt.want {
				t.Fatalf("classifyExecutionFailure() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyAttemptFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		execution *models.CanvasNodeExecution
		want      string
	}{
		{
			name: "aborted transaction with deadlock result message",
			err:  errors.New("ERROR: current transaction is aborted, commands ignored until end of transaction block"),
			execution: &models.CanvasNodeExecution{
				Result:        models.CanvasNodeExecutionResultFailed,
				ResultReason:  models.CanvasNodeExecutionResultReasonError,
				ResultMessage: "ERROR: deadlock detected (SQLSTATE 40P01)",
			},
			want: executorReasonDeadlock,
		},
		{
			name: "failed execution without process error",
			execution: &models.CanvasNodeExecution{
				Result:        models.CanvasNodeExecutionResultFailed,
				ResultReason:  models.CanvasNodeExecutionResultReasonError,
				ResultMessage: "request failed",
			},
			want: models.CanvasNodeExecutionResultReasonError,
		},
		{
			name: "process error takes precedence over execution state",
			err:  gorm.ErrRecordNotFound,
			execution: &models.CanvasNodeExecution{
				Result:        models.CanvasNodeExecutionResultFailed,
				ResultReason:  models.CanvasNodeExecutionResultReasonError,
				ResultMessage: "ERROR: deadlock detected (SQLSTATE 40P01)",
			},
			want: executorReasonNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyAttemptFailure(tt.err, tt.execution); got != tt.want {
				t.Fatalf("classifyAttemptFailure() = %q, want %q", got, tt.want)
			}
		})
	}
}
