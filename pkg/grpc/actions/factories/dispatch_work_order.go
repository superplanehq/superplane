package factories

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DispatchWorkOrder(ctx context.Context, organizationID string, req *pb.DispatchWorkOrderRequest) (*pb.DispatchWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	db := database.DB(ctx)
	resolvedFactory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	resolvedOrder, err := findWorkOrder(db, resolvedFactory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}
	orderID := resolvedOrder.ID
	hasPriorDispatch, priorErr := resolvedOrder.HasLineDispatch(db)
	runKind := "initial"
	if hasPriorDispatch {
		runKind = "rerun"
	}

	lineName := strings.TrimSpace(req.GetLineName())
	if lineName == "" {
		return nil, factoryErrorToStatus(invalidArgument("line_name is required"), "failed to dispatch work order")
	}

	var actor *uuid.UUID
	if userIDStr, ok := authentication.GetUserIdFromMetadata(ctx); ok {
		parsed, err := uuid.Parse(userIDStr)
		if err == nil {
			actor = &parsed
		}
	}

	line, err := resolvedFactory.FindLineByName(db, lineName)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	factory, order, err := DispatchWorkOrderOnLine(
		db,
		resolvedFactory,
		orderID,
		line,
		actor,
		int(req.GetStartStepIndex()),
		req.GetReplaceActive(),
		req.GetModel(),
		req.GetThinkingLevel(),
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}
	dispatchID := productanalytics.StableEventID(productanalytics.EventFactoryTaskStarted, order.ID.String(), order.UpdatedAt.UTC().String())
	if dispatch, findErr := order.FindActiveLineDispatch(db); findErr == nil {
		dispatchID = productanalytics.StableEventID(productanalytics.EventFactoryTaskStarted, dispatch.ID.String())
	}
	if priorErr != nil {
		log.WithError(priorErr).Warn("Skipped factory task started analytics")
	} else {
		captureFactoryProductEvent(
			ctx,
			db,
			factory,
			order.ID,
			productanalytics.EventFactoryTaskStarted,
			dispatchID,
			productanalytics.Properties{"run_kind": runKind},
		)
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	return &pb.DispatchWorkOrderResponse{
		Order: serialized,
	}, nil
}

type workOrderLineDispatchResult struct {
	factory      *models.Factory
	order        *models.FactoryWorkOrder
	startedSteps []*models.FactoryLineStepResult
	fromState    string
	logger       *log.Entry
}

func DispatchWorkOrderOnLine(
	db *gorm.DB,
	factory *models.Factory,
	orderID uuid.UUID,
	line *models.FactoryLine,
	actor *uuid.UUID,
	startIndex int,
	replaceActive bool,
	model string,
	thinkingLevel string,
) (*models.Factory, *models.FactoryWorkOrder, error) {
	var result *workOrderLineDispatchResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = dispatchWorkOrderOnLineTx(tx, factory, orderID, line, actor, startIndex, replaceActive, model, thinkingLevel)
		return err
	})
	if err != nil {
		return nil, nil, err
	}

	publishDispatchedWorkOrder(
		result.logger,
		result.factory.OrganizationID,
		result.factory.ID,
		result.order,
		actor,
		result.fromState,
		result.startedSteps,
	)
	return result.factory, result.order, nil
}

func dispatchWorkOrderOnLineTx(
	tx *gorm.DB,
	factory *models.Factory,
	orderID uuid.UUID,
	line *models.FactoryLine,
	actor *uuid.UUID,
	startIndex int,
	replaceActive bool,
	model string,
	thinkingLevel string,
) (*workOrderLineDispatchResult, error) {
	if factory == nil || line == nil || orderID == uuid.Nil {
		return nil, invalidArgument("work order dispatch is incomplete")
	}

	orgID := factory.OrganizationID
	if err := models.LockOrganizationTaskAdmission(tx, orgID); err != nil {
		return nil, err
	}
	factoryID := factory.ID
	lineID := line.ID

	f, err := models.FindFactory(tx, orgID, factoryID)
	if err != nil {
		return nil, err
	}
	factory = f

	order, err := factory.FindWorkOrder(tx, orderID)
	if err != nil {
		return nil, err
	}
	if err := order.LockForUpdate(tx); err != nil {
		return nil, err
	}

	logger := logging.WithWorkOrder(logging.ForFactory(*factory), *order).
		WithField("organization_id", factory.OrganizationID)
	if !order.IsDispatchable() {
		return nil, models.ErrFactoryWorkOrderNotDispatchable
	}

	currentLine, err := factory.FindLine(tx, lineID)
	if err != nil {
		return nil, err
	}

	if len(currentLine.Steps) == 0 {
		return nil, models.ErrFactoryLineHasNoSteps
	}

	model = strings.TrimSpace(model)
	if model != "" {
		allowed, err := listLineRunnerModels(tx, orgID, factoryID, currentLine.Name)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(allowed, model) {
			return nil, invalidArgument("model is not available on this line")
		}
	}
	normalizedThinking, err := runner.NormalizeDispatchThinkingLevel(thinkingLevel)
	if err != nil {
		return nil, invalidArgument(err.Error())
	}

	fromState := order.State
	if err := fillMissingWorkOrderTitle(tx, order); err != nil {
		return nil, err
	}
	if _, err := order.ClearAutoStart(tx); err != nil {
		return nil, err
	}

	var startedSteps []*models.FactoryLineStepResult
	if replaceActive && startIndex > 0 {
		_, started, err := order.RetryLineStep(tx, currentLine, startIndex)
		if err != nil {
			return nil, err
		}
		startedSteps = started
		if err := applyDispatchedWorkOrderState(tx, order, actor, startedSteps); err != nil {
			return nil, err
		}
		return &workOrderLineDispatchResult{
			factory:      factory,
			order:        order,
			startedSteps: startedSteps,
			fromState:    fromState,
			logger:       logger,
		}, nil
	}

	var abandoned []*models.FactoryLineStepResult
	_, err = order.FindActiveLineDispatch(tx)
	if err == nil {
		if !replaceActive {
			return nil, models.ErrFactoryWorkOrderLineDispatchActive
		}
		abandoned, err = order.AbandonActiveLineDispatch(tx)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	_, result, err := currentLine.DispatchFromWithModel(tx, order, startIndex, model, normalizedThinking)
	if err != nil {
		return nil, err
	}

	if err := applyDispatchedWorkOrderState(tx, order, actor, []*models.FactoryLineStepResult{result}); err != nil {
		return nil, err
	}

	return &workOrderLineDispatchResult{
		factory:      factory,
		order:        order,
		startedSteps: append(abandoned, result),
		fromState:    fromState,
		logger:       logger,
	}, nil
}

// A queued start stays a draft so Backlog pagination and filters agree with
// the board. Assign its owner now; admission opens it when execution starts.
func applyDispatchedWorkOrderState(tx *gorm.DB, order *models.FactoryWorkOrder, actor *uuid.UUID, results []*models.FactoryLineStepResult) error {
	for _, result := range results {
		if result != nil && result.Execution != nil && result.Execution.WorkOrderID == order.ID {
			return order.TransitionOnDispatch(tx, actor)
		}
	}
	if order.State == models.FactoryWorkOrderStateDraft && actor != nil {
		return order.UpdateAssignees(tx, []uuid.UUID{*actor}, *actor)
	}
	return nil
}

func publishDispatchedWorkOrder(
	logger *log.Entry,
	orgID uuid.UUID,
	factoryID uuid.UUID,
	order *models.FactoryWorkOrder,
	actor *uuid.UUID,
	fromState string,
	startedSteps []*models.FactoryLineStepResult,
) {
	for _, result := range startedSteps {
		if result != nil && result.Cancellation != nil {
			cancellation := result.Cancellation
			messages.PublishRunCancellationDrain(cancellation.Run.WorkflowID, cancellation.Drain)
			if err := messages.NewCanvasRunMessage(cancellation.Run.WorkflowID.String(), cancellation.Run.ID.String()).Publish(); err != nil {
				logger.WithError(err).Warn("Failed to publish replaced run cancellation")
			}
		}
		publishPendingStepRun(logger, result)
	}

	publishedOrders := map[uuid.UUID]struct{}{order.ID: {}}
	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		order.ID.String(),
		factoryevents.EventTypeLineStepExecutionCreated,
	); err != nil {
		logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
	for _, result := range startedSteps {
		if result == nil {
			continue
		}
		var extraID, extraFactoryID uuid.UUID
		if result.Execution != nil {
			extraID, extraFactoryID = result.Execution.WorkOrderID, result.Execution.FactoryID
		} else if result.QueueItem != nil {
			extraID, extraFactoryID = result.QueueItem.WorkOrderID, result.QueueItem.FactoryID
		} else {
			continue
		}
		if _, seen := publishedOrders[extraID]; seen {
			continue
		}
		publishedOrders[extraID] = struct{}{}
		if err := messages.PublishFactoryWorkOrderUpdated(
			extraFactoryID.String(),
			extraID.String(),
			factoryevents.EventTypeLineStepExecutionCreated,
		); err != nil {
			logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", extraID)
		}
	}

	if fromState != order.State {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: orgID.String(),
			FactoryID:      factoryID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      fromState,
			ToState:        order.State,
		}
		if actor != nil {
			notification.ActorUserID = actor.String()
		}
		if err := notification.Publish(); err != nil {
			logger.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
		}
	}
}

// publishPendingStepRun publishes one started step and logs that step's
// work order. A replace or retry can also start a queued run for another
// order, so the dispatch order on logger is not the owner of every run.
func publishPendingStepRun(logger *log.Entry, result *models.FactoryLineStepResult) {
	if result == nil || result.Run == nil {
		return
	}

	runLogger := logger.WithField("run_id", result.Run.ID)
	if result.Execution != nil && result.Execution.WorkOrderID != uuid.Nil {
		runLogger = runLogger.WithField("order_id", result.Execution.WorkOrderID)
	}

	if err := messages.NewCanvasRunMessage(result.Run.WorkflowID.String(), result.Run.ID.String()).PublishPending(); err != nil {
		runLogger.WithError(err).Errorf("Error publishing pending canvas run message: %v", err)
		return
	}
	runLogger.Info("Published pending canvas run")
}
