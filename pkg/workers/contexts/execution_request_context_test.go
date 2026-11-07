package contexts

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__ExecutionRequestContext_DoesNotScheduleRequestWhenPersistedExecutionIsFinished(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	ctx := NewExecutionRequestContext(database.Conn(), execution)
	err := ctx.ScheduleActionCall("poll", map[string]any{}, time.Second)
	require.NoError(t, err)

	pending, err := models.CountPendingRequestsForExecutionsInTransaction(database.Conn(), []uuid.UUID{execution.ID})
	require.NoError(t, err)
	assert.Zero(t, pending)
}

func Test__ExecutionRequestContext_SchedulesWhileMatchingRequestIsLocked(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		transactional bool
		rollback      bool
		parameters    map[string]any
	}{
		{name: "autocommit", parameters: map[string]any{"ref": "main", "attempt": float64(1)}},
		{name: "transaction commit", transactional: true, parameters: map[string]any{"attempt": float64(1), "ref": "main"}},
		{name: "transaction rollback with nil parameters", transactional: true, rollback: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.Conn()
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
			event := support.EmitCanvasEventForNode(t, canvas.ID, "checks", "default", nil)
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, "checks", event.ID, event.ID)
			otherExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "checks", event.ID, event.ID)
			parameters := scenario.parameters
			if parameters == nil {
				parameters = map[string]any{}
			}
			oldTime := time.Now().Add(-time.Minute)
			createRequest := func(owner *models.CanvasNodeExecution, action string, params map[string]any, requestType, state string) *models.CanvasNodeRequest {
				request := &models.CanvasNodeRequest{
					ID: uuid.New(), WorkflowID: canvas.ID, NodeID: "checks", ExecutionID: &owner.ID,
					Type: requestType, State: state, RunAt: oldTime, CreatedAt: oldTime, UpdatedAt: oldTime,
					Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
						InvokeAction: &models.InvokeAction{ActionName: action, Parameters: params},
					}),
				}
				require.NoError(t, db.Create(request).Error)
				return request
			}
			held := createRequest(execution, "evaluate", parameters, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestStatePending)
			matching := createRequest(execution, "evaluate", parameters, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestStatePending)
			controls := []*models.CanvasNodeRequest{
				createRequest(otherExecution, "evaluate", parameters, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestStatePending),
				createRequest(execution, "other", parameters, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestStatePending),
				createRequest(execution, "evaluate", map[string]any{"ref": "other"}, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestStatePending),
				createRequest(execution, "evaluate", parameters, models.NodeRequestTypeVisualEvidenceCapture, models.NodeExecutionRequestStatePending),
				createRequest(execution, "evaluate", parameters, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestStateCompleted),
			}

			// Acquire the worker's row lock before scheduling, and keep it until scheduling finishes.
			worker := db.Begin()
			require.NoError(t, worker.Error)
			defer worker.Rollback()
			_, err := models.LockNodeRequestNow(worker, held.ID)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			scheduler := db.WithContext(ctx)
			if scenario.transactional {
				scheduler = scheduler.Begin()
				require.NoError(t, scheduler.Error)
				defer scheduler.Rollback()
				// A worker must still complete a matching request that its own transaction holds.
				_, err = models.LockNodeRequestNow(scheduler, matching.ID)
				require.NoError(t, err)
			}
			before := time.Now()
			delay := 7 * time.Second
			err = NewExecutionRequestContext(scheduler, execution).ScheduleActionCall("evaluate", scenario.parameters, delay)
			require.NoError(t, err, "scheduling must finish while another worker holds a matching row")
			after := time.Now()

			assertRequest := func(conn *gorm.DB, original *models.CanvasNodeRequest, state string) {
				t.Helper()
				stored, err := models.FindCanvasNodeRequest(conn, original.ID)
				require.NoError(t, err)
				assert.Equal(t, state, stored.State)
				if state == original.State {
					assert.WithinDuration(t, original.UpdatedAt, stored.UpdatedAt, time.Microsecond)
					return
				}
				assert.False(t, stored.UpdatedAt.Before(before.Add(-time.Microsecond)))
			}
			assertRequest(db, held, models.NodeExecutionRequestStatePending)
			assertRequest(scheduler, matching, models.NodeExecutionRequestStateCompleted)
			for _, control := range controls {
				assertRequest(scheduler, control, control.State)
			}
			var replacement models.CanvasNodeRequest
			require.NoError(t, scheduler.Where("execution_id = ? AND id NOT IN ? AND state = ? AND type = ?", execution.ID,
				[]uuid.UUID{held.ID, matching.ID, controls[1].ID, controls[2].ID}, models.NodeExecutionRequestStatePending, models.NodeRequestTypeInvokeAction).
				First(&replacement).Error)
			assert.Equal(t, "evaluate", replacement.Spec.Data().InvokeAction.ActionName)
			assert.EqualValues(t, parameters, replacement.Spec.Data().InvokeAction.Parameters)
			assert.False(t, replacement.RunAt.Before(before.Add(delay).Add(-time.Microsecond)))
			assert.False(t, replacement.RunAt.After(after.Add(delay)))

			if scenario.transactional {
				assertRequest(db, matching, models.NodeExecutionRequestStatePending)
				_, err := models.FindCanvasNodeRequest(db, replacement.ID)
				require.ErrorIs(t, err, gorm.ErrRecordNotFound)
				if scenario.rollback {
					require.NoError(t, scheduler.Rollback().Error)
					assertRequest(db, matching, models.NodeExecutionRequestStatePending)
					_, err = models.FindCanvasNodeRequest(db, replacement.ID)
					require.ErrorIs(t, err, gorm.ErrRecordNotFound)
				} else {
					require.NoError(t, scheduler.Commit().Error)
					assertRequest(db, matching, models.NodeExecutionRequestStateCompleted)
					_, err = models.FindCanvasNodeRequest(db, replacement.ID)
					require.NoError(t, err)
				}
			}
			require.NoError(t, held.Complete(worker))
			require.NoError(t, worker.Commit().Error)
			assertRequest(db, held, models.NodeExecutionRequestStateCompleted)
		})
	}
}
