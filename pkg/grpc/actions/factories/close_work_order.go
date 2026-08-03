package factories

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CloseWorkOrder(ctx context.Context, organizationID string, req *pb.CloseWorkOrderRequest) (*pb.CloseWorkOrderResponse, error) {
	closeLog := log.WithFields(log.Fields{
		"organization_id": organizationID,
		"factory_id":      req.GetFactoryId(),
		"order_id":        req.GetOrderId(),
		"result":          req.GetResult().String(),
	})

	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		closeLog.WithField("step", "parse_organization_id").WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		closeLog.WithField("step", "parse_factory_id").WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		closeLog.WithField("step", "parse_order_id").WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	result, err := closeWorkOrderResult(req.GetResult())
	if err != nil {
		closeLog.WithField("step", "parse_result").WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	closeLog = closeLog.WithField("parsed_result", result)

	order, err := models.CloseFactoryWorkOrder(database.DB(ctx), orgID, factoryID, orderID, result)
	if err != nil {
		closeLog.WithField("step", "close_factory_work_order").WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	closeLog.WithFields(log.Fields{
		"step":         "close_factory_work_order",
		"order_state":  order.State,
		"order_result": order.Result,
	}).Info("close work order persisted")

	serialized, err := loadAndSerializeWorkOrder(ctx, order)
	if err != nil {
		closeLog.WithField("step", "load_and_serialize_work_order").WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	return &pb.CloseWorkOrderResponse{
		Order: serialized,
	}, nil
}
