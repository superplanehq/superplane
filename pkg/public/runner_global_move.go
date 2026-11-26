package public

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func (s *Server) handleRunnerMoveTask(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeTasksMove)
	if !ok {
		return
	}
	var request struct {
		Task   *string `json:"task"`
		Column *string `json:"column"`
		Next   *bool   `json:"next"`
	}
	if err := decodeRunnerJSON(w, r, &request, runnerTaskMutationBodyMaxBytes); err != nil {
		return
	}
	if (request.Column == nil) == (request.Next == nil) || (request.Next != nil && !*request.Next) || (request.Column != nil && strings.TrimSpace(*request.Column) == "") {
		http.Error(w, "Pass exactly one of column or next: true", http.StatusBadRequest)
		return
	}
	column := ""
	if request.Column != nil {
		column = strings.TrimSpace(*request.Column)
	}
	db := database.DB(r.Context())
	var target *taskOwnerTarget
	waitingStage := ""
	if scope.Flow == runneraction.RunnerFlowSchedule {
		target, waitingStage, ok = s.runnerWaitingTaskTarget(w, r, scope, request.Task)
	} else {
		target, ok = s.runnerTaskTarget(w, r, scope)
		if ok && request.Task != nil {
			own, err := target.factory.FindWorkOrderByRef(db, *request.Task)
			if err != nil || own.ID != target.order.ID {
				http.Error(w, "A task run can move only its own task. Remove task", http.StatusBadRequest)
				return
			}
		}
	}
	if !ok {
		return
	}
	fromState := target.order.State
	var leftColumn string
	var backlogEvents []models.CanvasEvent
	var result *models.FactoryTaskMoveResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		leftColumn, err = models.WorkOrderBoardColumn(tx, target.order)
		if err != nil {
			return err
		}
		if waitingStage != "" {
			result, err = target.order.MoveWaitingFromRun(tx, target.canvasRunID, waitingStage, column, request.Next != nil)
		} else {
			result, err = target.order.MoveFromRun(tx, target.canvasRunID, column, request.Next != nil)
		}
		if err != nil {
			return err
		}
		if result.Changed && target.order.State == models.FactoryWorkOrderStateDraft {
			backlogEvents = contexts.EmitBacklogCustomAutomations(tx, target.factory, target.order)
		}
		if !result.Changed || !target.order.IsOpen() {
			return nil
		}
		dispatch, err := target.order.FindActiveLineDispatch(tx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, step := range dispatch.Steps {
			if step.StageKey != result.Column {
				continue
			}
			blocked, err := factoryactions.ImplementStepBlockedByMissingBranch(tx, target.factory, target.order, step.AppID)
			if err != nil {
				return err
			}
			if blocked {
				return fmt.Errorf("%w: set the default branch before entering Implement", models.ErrFactoryTaskMoveInvalid)
			}
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, models.ErrFactoryTaskMoveInvalid):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, models.ErrFactoryTaskNotWaiting):
			http.Error(w, "Task is not waiting in this stage", http.StatusConflict)
		case errors.Is(err, models.ErrFactoryTaskMoveStale), errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound), errors.Is(err, models.ErrFactoryWorkOrderLineDispatchNotFound):
			http.Error(w, "Calling run has no current task stage", http.StatusConflict)
		case errors.As(err, &pgErr) && (pgErr.Code == "55P03" || pgErr.Code == "40P01"):
			http.Error(w, "Task stage is changing. Retry the request", http.StatusConflict)
		default:
			writeRunnerTaskError(w, err)
		}
		return
	}
	if result.Changed {
		contexts.PublishWorkOrderCreatedEvents(target.factory.OrganizationID, backlogEvents)
		for _, stop := range result.Cancellations {
			if stop == nil || stop.Run == nil {
				continue
			}
			messages.PublishRunCancellationDrain(stop.Run.WorkflowID, stop.Drain)
			if stop.NewlyCancelling {
				if err := messages.NewCanvasRunMessage(stop.Run.WorkflowID.String(), stop.Run.ID.String()).Publish(); err != nil {
					log.WithError(err).Warn("Failed to publish superseded run cancellation")
				}
			}
		}
		for _, step := range result.Started {
			if step.Run != nil {
				if err := messages.NewCanvasRunMessage(step.Run.WorkflowID.String(), step.Run.ID.String()).PublishPending(); err != nil {
					log.WithError(err).Warn("Failed to publish moved task run")
				}
			}
			if step.Execution != nil {
				if err := messages.PublishFactoryWorkOrderUpdated(step.Execution.FactoryID.String(), step.Execution.WorkOrderID.String(), factoryevents.EventTypeLineStepExecutionCreated); err != nil {
					log.WithError(err).Warn("Failed to publish admitted task update")
				}
			}
		}
		left := leftColumn
		for _, entered := range result.EnteredColumns {
			if err := factoryactions.StartColumnLeaveCustomAutomations(db, target.factory, target.order, left, target.canvasRunID.String()); err != nil {
				log.WithError(err).Warn("Failed to start stage leave automation")
			}
			if err := factoryactions.StartAutomaticTaskColumnCustomAutomations(db, target.factory, target.order, entered); err != nil {
				log.WithError(err).Warn("Failed to start stage enter automation")
			}
			left = entered
		}
		publishRunnerTaskUpdate(target.factory, target.order, factoryevents.EventTypeOrderStatusUpdated)
		if fromState != target.order.State {
			notification := messages.FactoryWorkOrderNotificationMessage{OrganizationID: target.factory.OrganizationID.String(), FactoryID: target.factory.ID.String(), OrderID: target.order.ID.String(), EventType: factoryevents.EventTypeOrderStatusUpdated, FromState: fromState, ToState: target.order.State, Result: target.order.Result}
			if err := notification.Publish(); err != nil {
				log.WithError(err).Warn("Failed to publish moved task notification")
			}
		}
	}
	_ = writeJSON(w, http.StatusOK, struct {
		runnerTaskView
		FromColumn string `json:"from_column"`
		Column     string `json:"column"`
		Changed    bool   `json:"changed"`
	}{runnerTaskViewFrom(s.BaseURL, target.factory, target.order), result.FromColumn, result.Column, result.Changed})
}

// A run with no task, such as a webhook run, can move only a task that waits
// in the stage of its own automation.
func (s *Server) runnerWaitingTaskTarget(w http.ResponseWriter, r *http.Request, scope *runneraction.RunnerScope, ref *string) (*taskOwnerTarget, string, bool) {
	if ref == nil || strings.TrimSpace(*ref) == "" {
		http.Error(w, "Pass task. This run has no task", http.StatusBadRequest)
		return nil, "", false
	}
	db := database.DB(r.Context())
	factory, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		writeRunnerTaskError(w, err)
		return nil, "", false
	}
	order, err := factory.FindWorkOrderByRef(db, *ref)
	if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
		http.Error(w, "Task not found", http.StatusNotFound)
		return nil, "", false
	}
	if err != nil {
		writeRunnerTaskError(w, err)
		return nil, "", false
	}
	canvas, err := models.FindCanvasInTransaction(db, scope.OrganizationID, scope.CanvasID)
	if err != nil {
		writeRunnerTaskError(w, err)
		return nil, "", false
	}
	if canvas.ColumnKey == nil || *canvas.ColumnKey == "" {
		http.Error(w, "This automation has no stage", http.StatusConflict)
		return nil, "", false
	}
	target, ok := s.openTaskOwnerTarget(w, r, scope.OrganizationID, scope.FactoryID, order.ID)
	if !ok {
		return nil, "", false
	}
	target.canvasID = scope.CanvasID
	target.canvasRunID = scope.CanvasRunID
	stage := *canvas.ColumnKey
	if stage == models.CanvasColumnKeyImplement {
		stage = "phase-0"
	}
	return target, stage, true
}
