package factories

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// ReorderWorkOrder moves a work order to a new manual position, expressed
// as the two work orders that should end up next to it (see
// pb.ReorderWorkOrderRequest). Both neighbors, when given, must resolve to
// the same board column as the moved order — reordering never moves a
// card between columns.
func ReorderWorkOrder(
	ctx context.Context,
	organizationID string,
	req *pb.ReorderWorkOrderRequest,
) (*pb.ReorderWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to reorder work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to reorder work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to reorder work order")
	}

	db := database.DB(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		f, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := f.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		previous, err := findOptionalWorkOrderNeighbor(tx, f, req.PreviousOrderId)
		if err != nil {
			return err
		}

		next, err := findOptionalWorkOrderNeighbor(tx, f, req.NextOrderId)
		if err != nil {
			return err
		}

		if err := ensureSameWorkOrderLane(tx, order, previous, next); err != nil {
			return err
		}

		return order.Reorder(tx, previous, next)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to reorder work order")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factoryevents.EventTypeOrderUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	f, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to reorder work order")
	}

	order, err := f.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to reorder work order")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, f, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to reorder work order")
	}

	return &pb.ReorderWorkOrderResponse{Order: serialized}, nil
}

// findOptionalWorkOrderNeighbor resolves an optional neighbor id, scoped to
// the same factory as the order being moved. A nil/empty id means "no
// neighbor on that side" (move to the top or bottom of the column).
func findOptionalWorkOrderNeighbor(tx *gorm.DB, f *models.Factory, id *string) (*models.FactoryWorkOrder, error) {
	if id == nil || *id == "" {
		return nil, nil
	}

	neighborID, err := parseOrderID(*id)
	if err != nil {
		return nil, err
	}

	return f.FindWorkOrder(tx, neighborID)
}

// ensureSameWorkOrderLane rejects a reorder whose neighbors would place the
// order in a different board column, mirroring the frontend's per-lane drop
// constraint on the server.
func ensureSameWorkOrderLane(tx *gorm.DB, order, previous, next *models.FactoryWorkOrder) error {
	lane, err := workOrderLane(tx, order)
	if err != nil {
		return err
	}

	for _, neighbor := range []*models.FactoryWorkOrder{previous, next} {
		if neighbor == nil {
			continue
		}
		neighborLane, err := workOrderLane(tx, neighbor)
		if err != nil {
			return err
		}
		if neighborLane != lane {
			return models.ErrFactoryWorkOrderLaneMismatch
		}
	}

	return nil
}

func workOrderLane(tx *gorm.DB, order *models.FactoryWorkOrder) (models.FactoryWorkOrderLane, error) {
	hasActiveDispatch := false
	if order.State == models.FactoryWorkOrderStateOpen {
		_, err := order.FindActiveLineDispatch(tx)
		switch {
		case err == nil:
			hasActiveDispatch = true
		case errors.Is(err, gorm.ErrRecordNotFound):
			hasActiveDispatch = false
		default:
			return "", err
		}
	}

	return order.Lane(hasActiveDispatch), nil
}
