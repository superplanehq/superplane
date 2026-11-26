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
	"gorm.io/gorm"
)

func (s *Server) handleRunnerMoveTask(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopeTasksMove)(w, r)
	if !ok {
		return
	}
	var request struct {
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
	fromState := target.order.State
	var result *models.FactoryTaskMoveResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = target.order.MoveFromRun(tx, target.canvasRunID, column, request.Next != nil)
		if err != nil {
			return err
		}
		if !result.Changed || !target.order.IsOpen() {
			return nil
		}
		execution, err := models.FindWorkOrderExecutionForRun(tx, target.canvasRunID)
		if err != nil {
			return err
		}
		dispatch, err := models.FindWorkOrderLineDispatch(tx, execution.LineDispatchID)
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
		left := result.FromColumn
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
