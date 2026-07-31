package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
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

	tx := database.DB(ctx)
	var order *models.FactoryWorkOrder
	var pendingRun *models.CanvasRun

	err = tx.Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		order, dispatchErr = models.FindFactoryWorkOrder(tx, orgID, factoryID, orderID)
		if dispatchErr != nil {
			return dispatchErr
		}

		if order.State != models.FactoryWorkOrderStateOpen {
			return models.ErrFactoryWorkOrderNotOpen
		}

		line, dispatchErr := models.FindFactoryLineByName(tx, orgID, factoryID, lineName)
		if dispatchErr != nil {
			return dispatchErr
		}

		_, dispatchErr = models.FindActiveFactoryWorkOrderExecution(tx, order.ID, line.ID)
		if dispatchErr == nil {
			return models.ErrFactoryWorkOrderLineActive
		}
		if !errors.Is(dispatchErr, models.ErrFactoryWorkOrderExecutionNotFound) {
			return dispatchErr
		}

		if len(line.Steps) == 0 {
			return models.ErrFactoryLineHasNoSteps
		}

		result, dispatchErr := line.StartStep(tx, order, 0)
		if dispatchErr != nil {
			return dispatchErr
		}

		pendingRun = result.Run
		return nil
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to dispatch work order")
	}

	if pendingRun != nil {
		if err := messages.NewCanvasRunMessage(pendingRun.WorkflowID.String(), pendingRun.ID.String()).PublishPending(); err != nil {
			return nil, factoryErrorToStatus(err, "failed to dispatch work order")
		}
	}

	return &pb.DispatchWorkOrderResponse{
		Order: serializeWorkOrder(order),
	}, nil
}
