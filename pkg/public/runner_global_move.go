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
		Task     *string `json:"task"`
		Stage    *string `json:"stage"`
		Column   *string `json:"column"`
		Next     *bool   `json:"next"`
		Previous *bool   `json:"previous"`
	}
	if err := decodeRunnerJSON(w, r, &request, runnerTaskMutationBodyMaxBytes); err != nil {
		return
	}
	// column is the earlier name of stage.
	if request.Stage == nil {
		request.Stage = request.Column
	}
	to, ok := runnerMoveDestination(request.Stage, request.Next, request.Previous)
	if !ok {
		http.Error(w, "Pass exactly one of stage, next: true, or previous: true", http.StatusBadRequest)
		return
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
		if ok && scope.Flow == runneraction.RunnerFlowTask {
			waitingStage, ok = s.runnerTaskSideRunStage(w, db, scope)
		}
	}
	if !ok {
		return
	}
	fromState := target.order.State
	var leftStage string
	var backlogEvents []models.CanvasEvent
	var result *models.FactoryTaskMoveResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		leftStage, err = models.WorkOrderBoardColumn(tx, target.order)
		if err != nil {
			return err
		}
		if waitingStage != "" {
			result, err = target.order.MoveWaitingFromRun(tx, target.canvasRunID, waitingStage, to)
		} else {
			result, err = target.order.MoveFromRun(tx, target.canvasRunID, to)
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
			if step.StageKey != result.Stage {
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
		left := leftStage
		for _, entered := range result.EnteredStages {
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
		FromStage string `json:"from_stage"`
		Stage     string `json:"stage"`
		Changed   bool   `json:"changed"`
	}{runnerTaskViewFrom(s.BaseURL, target.factory, target.order), result.FromStage, result.Stage, result.Changed})
}

// handleRunnerAttachTask links a run with no task to a task that waits in the
// stage of its automation. The task timeline then shows the run in that stage.
func (s *Server) handleRunnerAttachTask(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeTasksAttach)
	if !ok {
		return
	}
	var request struct {
		Task *string `json:"task"`
	}
	if err := decodeRunnerJSON(w, r, &request, runnerTaskMutationBodyMaxBytes); err != nil {
		return
	}
	target, stage, ok := s.runnerWaitingTaskTarget(w, r, scope, request.Task)
	if !ok {
		return
	}
	db := database.DB(r.Context())
	err := db.Transaction(func(tx *gorm.DB) error {
		return target.order.AttachWaitingRun(tx, target.canvasRunID, stage)
	})
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, models.ErrFactoryTaskNotWaiting):
			http.Error(w, "Task is not waiting in this stage", http.StatusConflict)
		case errors.Is(err, models.ErrFactoryTaskMoveStale):
			http.Error(w, "Calling run is not running", http.StatusConflict)
		case errors.As(err, &pgErr) && (pgErr.Code == "55P03" || pgErr.Code == "40P01"):
			http.Error(w, "Task stage is changing. Retry the request", http.StatusConflict)
		default:
			writeRunnerTaskError(w, err)
		}
		return
	}
	publishRunnerTaskUpdate(target.factory, target.order, factoryevents.EventTypeOrderUpdated)
	view, err := s.runnerTaskDetail(db, target.factory, target.order)
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, view)
}

// runnerMoveDestination accepts exactly one of stage, next: true, or previous: true.
func runnerMoveDestination(stage *string, next, previous *bool) (models.FactoryTaskMoveTo, bool) {
	to := models.FactoryTaskMoveTo{Next: next != nil && *next, Previous: previous != nil && *previous}
	if stage != nil {
		to.Stage = strings.TrimSpace(*stage)
	}
	chosen := 0
	for _, set := range []bool{stage != nil, next != nil, previous != nil} {
		if set {
			chosen++
		}
	}
	if chosen != 1 || (next != nil && !to.Next) || (previous != nil && !to.Previous) || (stage != nil && to.Stage == "") {
		return to, false
	}
	return to, true
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
	stage, ok := runnerAutomationStage(w, db, scope)
	if !ok {
		return nil, "", false
	}
	target, ok := s.openTaskOwnerTarget(w, r, scope.OrganizationID, scope.FactoryID, order.ID)
	if !ok {
		return nil, "", false
	}
	target.canvasID = scope.CanvasID
	target.canvasRunID = scope.CanvasRunID
	return target, stage, true
}

// A task run that is not a line step, such as a manual automation, can move
// its task only while the task waits in the stage of its automation.
// It returns an empty stage for a line step run.
func (s *Server) runnerTaskSideRunStage(w http.ResponseWriter, db *gorm.DB, scope *runneraction.RunnerScope) (string, bool) {
	_, err := models.FindWorkOrderExecutionForRun(db, scope.CanvasRunID)
	if err == nil {
		return "", true
	}
	if !errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
		writeRunnerTaskError(w, err)
		return "", false
	}
	return runnerAutomationStage(w, db, scope)
}

func runnerAutomationStage(w http.ResponseWriter, db *gorm.DB, scope *runneraction.RunnerScope) (string, bool) {
	canvas, err := models.FindCanvasInTransaction(db, scope.OrganizationID, scope.CanvasID)
	if err != nil {
		writeRunnerTaskError(w, err)
		return "", false
	}
	if canvas.ColumnKey == nil || *canvas.ColumnKey == "" {
		http.Error(w, "This automation has no stage", http.StatusConflict)
		return "", false
	}
	if *canvas.ColumnKey == models.CanvasColumnKeyImplement {
		return "phase-0", true
	}
	return *canvas.ColumnKey, true
}
