package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func loadAndSerializeWorkOrder(ctx context.Context, order *models.FactoryWorkOrder) (*pb.WorkOrder, error) {
	executionsByOrderID, err := models.ListFactoryWorkOrderExecutionsByWorkOrderIDs(
		database.DB(ctx),
		[]uuid.UUID{order.ID},
	)
	if err != nil {
		return nil, err
	}

	return serializeWorkOrder(order, executionsByOrderID[order.ID]), nil
}

func loadAndSerializeWorkOrders(ctx context.Context, orders []models.FactoryWorkOrder) ([]*pb.WorkOrder, error) {
	workOrderIDs := make([]uuid.UUID, len(orders))
	for i := range orders {
		workOrderIDs[i] = orders[i].ID
	}

	executionsByOrderID, err := models.ListFactoryWorkOrderExecutionsByWorkOrderIDs(database.DB(ctx), workOrderIDs)
	if err != nil {
		return nil, err
	}

	result := make([]*pb.WorkOrder, len(orders))
	for i := range orders {
		result[i] = serializeWorkOrder(&orders[i], executionsByOrderID[orders[i].ID])
	}

	return result, nil
}
