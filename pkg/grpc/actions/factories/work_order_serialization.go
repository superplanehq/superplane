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
	dispatchesByOrderID, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, []uuid.UUID{order.ID})
	if err != nil {
		return nil, err
	}

	creatorAutomations, err := models.ResolveFactoryWorkOrderCreatorAutomations(db, []models.FactoryWorkOrder{*order})
	if err != nil {
		return nil, err
	}

	runsByOrder, err := loadPRFeedbackRunsByWorkOrderID(db, factory, []uuid.UUID{order.ID})
	if err != nil {
		return nil, err
	}

	serialized, err := serializeWorkOrder(
		factory,
		order,
		dispatchesByOrderID[order.ID],
		creatorAutomations[order.ID],
	)
	if err != nil {
		return nil, err
	}
	serialized.PrFeedbackRuns = runsByOrder[order.ID]
	return serialized, nil
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
	dispatchesByOrderID, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, workOrderIDs)
	if err != nil {
		return nil, err
	}

	creatorAutomations, err := models.ResolveFactoryWorkOrderCreatorAutomations(db, orders)
	if err != nil {
		return nil, err
	}

	runsByOrder, err := loadPRFeedbackRunsByWorkOrderID(db, factory, workOrderIDs)
	if err != nil {
		return nil, err
	}

	result := make([]*pb.WorkOrder, len(orders))
	for i := range orders {
		serialized, err := serializeWorkOrder(
			factory,
			&orders[i],
			dispatchesByOrderID[orders[i].ID],
			creatorAutomations[orders[i].ID],
		)
		if err != nil {
			return nil, err
		}
		serialized.PrFeedbackRuns = runsByOrder[orders[i].ID]
		result[i] = serialized
	}

	return result, nil
}
