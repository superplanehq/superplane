package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

// loadAndSerializeWorkOrder is used by every single-order response (create,
// describe, dispatch, close, status/assignee updates): unlike
// loadAndSerializeWorkOrders (list endpoints), it also loads and populates
// `reactions`, since doing so here only costs one extra query for a single
// order rather than a batched join across a whole page of results.
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

	reactions, err := order.ListReactions(db)
	if err != nil {
		return nil, err
	}

	var currentUserID uuid.UUID
	if userIDStr, ok := authentication.GetUserIdFromMetadata(ctx); ok {
		currentUserID, _ = uuid.Parse(userIDStr)
	}

	return serializeWorkOrder(
		factory,
		order,
		executionsByOrderID[order.ID],
		creatorAutomations[order.ID],
		reactions,
		currentUserID,
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
		// Scope decision: reactions are omitted from list responses (`nil`,
		// `uuid.Nil`) to avoid an extra join per row; see
		// loadAndSerializeWorkOrder for the single-order equivalent.
		result[i] = serializeWorkOrder(
			factory,
			&orders[i],
			executionsByOrderID[orders[i].ID],
			creatorAutomations[orders[i].ID],
			nil,
			uuid.Nil,
		)
	}

	return result, nil
}
