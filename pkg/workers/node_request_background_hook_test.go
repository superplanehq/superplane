package workers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type backgroundHookProbe struct {
	core.Action
	handle func(core.BackgroundHookContext) (core.BackgroundHookResult, error)
}

func (p *backgroundHookProbe) HandleBackgroundHook(ctx core.BackgroundHookContext) (core.BackgroundHookResult, error) {
	return p.handle(ctx)
}

func Test__NodeRequestExecutor_BackgroundHookLeaseLifecycle(t *testing.T) {
	for _, scenario := range []string{"complete", "schedule action", "retry error", "lease replaced", "cancelled", "cancelled concurrently", "panic"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			componentName := "background_probe_" + uuid.New().String()
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "background", Type: models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			}}, nil)
			event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "background", "default", nil, map[string]any{"input": "stored"})
			execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, "background", event.ID, event.ID, map[string]any{"policy": "resolved"})
			runAt := time.Now().Add(-time.Minute)
			require.NoError(t, execution.CreateRequest(db, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
				InvokeAction: &models.InvokeAction{ActionName: "refresh", Parameters: map[string]any{}},
			}, &runAt))
			var request models.CanvasNodeRequest
			require.NoError(t, db.Where("execution_id = ?", execution.ID).First(&request).Error)
			executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
			calls := 0
			replacementDeadline := time.Now().Add(10 * time.Minute)
			probe := &backgroundHookProbe{
				Action: impl.NewDummyAction(impl.DummyActionOptions{Name: componentName, Hooks: []core.Hook{{Name: "refresh", Type: core.HookTypeInternal}}}),
				handle: func(ctx core.BackgroundHookContext) (core.BackgroundHookResult, error) {
					calls++
					assert.Equal(t, execution.RunID, ctx.RunID)
					assert.Equal(t, execution.ID, ctx.ExecutionID)
					assert.Equal(t, map[string]any{"policy": "resolved"}, ctx.Configuration)
					assert.Equal(t, event.Data.Data(), ctx.Data)
					// External work can acquire both rows while the handler runs.
					require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
						var stored models.CanvasNodeRequest
						if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).First(&stored, "id = ?", request.ID).Error; err != nil {
							return err
						}
						_, err := models.FindNodeExecutionInTransaction(tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}), canvas.ID, execution.ID)
						return err
					}))
					if calls == 1 {
						require.NoError(t, executor.RunNow(request))
						assert.Equal(t, 1, calls, "the live lease must suppress concurrent invocation")
					}
					switch scenario {
					case "schedule action":
						return core.BackgroundHookResult{}, ctx.Requests.ScheduleActionCall("refresh", nil, 10*time.Minute)
					case "retry error":
						return core.BackgroundHookResult{}, errors.New("GitHub unavailable")
					case "panic":
						panic("interrupted hook")
					case "lease replaced":
						if calls == 1 {
							require.NoError(t, db.Model(&request).Update("lease_expires_at", time.Now().Add(-time.Second)).Error)
							require.NoError(t, executor.RunNow(request))
							assert.ErrorIs(t, ctx.Requests.ScheduleActionCall("refresh", nil, time.Hour), core.ErrBackgroundHookLeaseLost)
							assert.ErrorIs(t, ctx.ExecutionState.Emit("default", "background.done", []any{map[string]any{}}), core.ErrBackgroundHookLeaseLost)
							return core.BackgroundHookResult{}, nil
						}
						return core.BackgroundHookResult{RetryAfter: time.Until(replacementDeadline)}, nil
					case "cancelled":
						require.NoError(t, execution.RequestCancellation(db, nil))
					case "cancelled concurrently":
						return core.BackgroundHookResult{}, cancelDuringBackgroundEmission(t, db, execution, ctx.ExecutionState)
					}
					return core.BackgroundHookResult{}, ctx.ExecutionState.Emit("default", "background.done", []any{map[string]any{"published": true}})
				},
			}
			// Exercise the registration wrapper as well as the executor contract.
			r.Registry.Actions[componentName] = registry.NewPanicableAction(probe)
			before := time.Now()
			err := executor.Execute(request)
			if scenario == "retry error" || scenario == "panic" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			stored, err := models.FindCanvasNodeRequest(db, request.ID)
			require.NoError(t, err)
			assert.Nil(t, stored.LeaseID)
			if scenario == "retry error" || scenario == "panic" || scenario == "lease replaced" {
				assert.Equal(t, models.NodeExecutionRequestStatePending, stored.State)
				assert.False(t, stored.RunAt.Before(before.Add(time.Minute-time.Second)))
			} else {
				assert.Equal(t, models.NodeExecutionRequestStateCompleted, stored.State)
			}
			if scenario == "lease replaced" {
				assert.Equal(t, 2, calls)
				assert.WithinDuration(t, replacementDeadline, stored.RunAt, time.Second)
				var pending int64
				require.NoError(t, db.Model(&models.CanvasNodeRequest{}).Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).Count(&pending).Error)
				assert.EqualValues(t, 1, pending, "the expired owner cannot replace the successor's request")
			}
			if scenario == "schedule action" {
				var pending []models.CanvasNodeRequest
				require.NoError(t, db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).Find(&pending).Error)
				require.Len(t, pending, 1)
				assert.NotEqual(t, request.ID, pending[0].ID)
				assert.WithinDuration(t, before.Add(10*time.Minute), pending[0].RunAt, time.Second)
			}
			var outputCount int64
			require.NoError(t, db.Model(&models.CanvasEvent{}).Where("execution_id = ?", execution.ID).Count(&outputCount).Error)
			if scenario == "complete" {
				assert.EqualValues(t, 1, outputCount)
			} else {
				assert.Zero(t, outputCount)
			}
		})
	}
}

func cancelDuringBackgroundEmission(t *testing.T, db *gorm.DB, execution *models.CanvasNodeExecution, state core.ExecutionStateContext) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	locked := make(chan int, 1)
	continueCancellation := make(chan struct{})
	var release sync.Once
	defer release.Do(func() { close(continueCancellation) })
	cancelled := make(chan error, 1)
	go func() {
		cancelled <- db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SET LOCAL lock_timeout = '2s'").Error; err != nil {
				return err
			}
			run, err := models.LockCanvasRunInTransaction(tx, execution.RunID)
			if err != nil {
				return err
			}
			var connectionID int
			if err := tx.Raw("SELECT pg_backend_pid()").Scan(&connectionID).Error; err != nil {
				return err
			}
			locked <- connectionID
			select {
			case <-continueCancellation:
			case <-ctx.Done():
				return ctx.Err()
			}
			_, err = run.RequestCancellation(tx, nil)
			return err
		})
	}()
	var connectionID int
	select {
	case connectionID = <-locked:
	case err := <-cancelled:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("cancellation did not acquire the run")
	}
	emitted := make(chan error, 1)
	go func() { emitted <- state.Emit("default", "background.done", []any{map[string]any{}}) }()
	// Wait for the real database dependency, rather than relying on scheduling.
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid)))", connectionID).Scan(&waiting).Error
		return err == nil && waiting
	}, 2*time.Second, 5*time.Millisecond)
	release.Do(func() { close(continueCancellation) })
	require.NoError(t, <-cancelled, "cancellation and output emission must use the same lock order")
	return <-emitted
}
