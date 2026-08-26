package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DispatchWorkOrder(ctx context.Context, organizationID string, req *pb.DispatchWorkOrderRequest) (*pb.DispatchWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
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

	var factory *models.Factory
	var order *models.FactoryWorkOrder
	var pendingRuns []*models.CanvasRun
	var startedSteps []*models.FactoryLineStepResult
	var logger *log.Entry
	var fromState string

	db := database.DB(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		f, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}
		factory = f

		order, err = factory.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		logger = logging.WithWorkOrder(logging.ForFactory(*factory), *order)
		if !order.IsDispatchable() {
			return models.ErrFactoryWorkOrderNotDispatchable
		}

		line, err := factory.FindLineByName(tx, lineName)
		if err != nil {
			return err
		}

		if len(line.Steps) == 0 {
			return models.ErrFactoryLineHasNoSteps
		}

		startIndex := int(req.GetStartStepIndex())
		fromState = order.State
		if err := order.TransitionOnDispatch(tx, actor); err != nil {
			return err
		}

		if req.GetReplaceActive() && startIndex > 0 {
			_, started, err := order.RetryLineStep(tx, line, startIndex)
			if err != nil {
				return err
			}
			startedSteps = started
			pendingRuns = pendingRunsFromStepResults(started)
			return nil
		}

		var abandoned []*models.FactoryLineStepResult
		_, err = order.FindActiveLineDispatch(tx)
		if err == nil {
			if !req.GetReplaceActive() {
				return models.ErrFactoryWorkOrderLineDispatchActive
			}
			abandoned, err = order.AbandonActiveLineDispatch(tx)
			if err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		_, result, err := line.DispatchFrom(tx, order, startIndex)
		if err != nil {
			return err
		}

		startedSteps = append(abandoned, result)
		pendingRuns = pendingRunsFromStepResults(startedSteps)
		return nil
	})

	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	for _, pendingRun := range pendingRuns {
		if err := messages.NewCanvasRunMessage(pendingRun.WorkflowID.String(), pendingRun.ID.String()).PublishPending(); err != nil {
			logger.WithError(err).Errorf("Error publishing pending canvas run message: %v", err)
		}
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
		if result == nil || result.Execution == nil {
			continue
		}
		extraID := result.Execution.WorkOrderID
		if _, seen := publishedOrders[extraID]; seen {
			continue
		}
		publishedOrders[extraID] = struct{}{}
		if err := messages.PublishFactoryWorkOrderUpdated(
			factoryID.String(),
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

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	return &pb.DispatchWorkOrderResponse{
		Order: serialized,
	}, nil
}

func pendingRunsFromStepResults(results []*models.FactoryLineStepResult) []*models.CanvasRun {
	var runs []*models.CanvasRun
	for _, result := range results {
		if result != nil && result.Run != nil {
			runs = append(runs, result.Run)
		}
	}
	return runs
}
