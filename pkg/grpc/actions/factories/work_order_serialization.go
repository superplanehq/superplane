package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func loadAndSerializeWorkOrder(ctx context.Context, factory *models.Factory, order *models.FactoryWorkOrder) (*pb.WorkOrder, error) {
	db := database.DB(ctx)
	executionsByOrderID, err := models.ListFactoryWorkOrderExecutionsByWorkOrderIDs(db, []uuid.UUID{order.ID})
	if err != nil {
		return nil, err
	}

	creatorAutomations, err := models.ResolveFactoryWorkOrderCreatorAutomations(db, []models.FactoryWorkOrder{*order})
	if err != nil {
		return nil, err
	}

	return serializeWorkOrder(
		factory,
		order,
		executionsByOrderID[order.ID],
		creatorAutomations[order.ID],
	), nil
}

func loadAndSerializeWorkOrders(ctx context.Context, factory *models.Factory, orders []models.FactoryWorkOrder) ([]*pb.WorkOrder, error) {
	if len(orders) == 0 {
		return nil, nil
	}

	workOrderIDs := make([]uuid.UUID, len(orders))
	for i := range orders {
		workOrderIDs[i] = orders[i].ID
	}

	db := database.DB(ctx)
	executionsByOrderID, err := models.ListFactoryWorkOrderExecutionsByWorkOrderIDs(db, workOrderIDs)
	if err != nil {
		return nil, err
	}

	creatorAutomations, err := models.ResolveFactoryWorkOrderCreatorAutomations(db, orders)
	if err != nil {
		return nil, err
	}

	result := make([]*pb.WorkOrder, len(orders))
	for i := range orders {
		result[i] = serializeWorkOrder(
			factory,
			&orders[i],
			executionsByOrderID[orders[i].ID],
			creatorAutomations[orders[i].ID],
		)
	}

	return result, nil
}
