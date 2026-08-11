package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func loadAndSerializeWorkOrder(ctx context.Context, order *models.FactoryWorkOrder) (*pb.WorkOrder, error) {
	db := database.DB(ctx)
	executionsByOrderID, err := models.ListFactoryWorkOrderExecutionsByWorkOrderIDs(db, []uuid.UUID{order.ID})
	if err != nil {
		return nil, err
	}

	createdByAutomation, err := resolveWorkOrderCreatorAutomation(db, order)
	if err != nil {
		return nil, err
	}

	return serializeWorkOrder(
		order,
		executionsByOrderID[order.ID],
		createdByAutomation,
	), nil
}

func loadAndSerializeWorkOrders(ctx context.Context, orders []models.FactoryWorkOrder) ([]*pb.WorkOrder, error) {
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

	result := make([]*pb.WorkOrder, len(orders))
	for i := range orders {
		createdByAutomation, err := resolveWorkOrderCreatorAutomation(db, &orders[i])
		if err != nil {
			return nil, err
		}
		result[i] = serializeWorkOrder(
			&orders[i],
			executionsByOrderID[orders[i].ID],
			createdByAutomation,
		)
	}

	return result, nil
}

// resolveWorkOrderCreatorAutomation resolves the automation that created the
// order (from its SourceRunID). Returns nil when the order wasn't created by
// an automation, when the source run/node is gone, or when the node isn't
// linked to a factory automation.
func resolveWorkOrderCreatorAutomation(tx *gorm.DB, order *models.FactoryWorkOrder) (*factory.AutomationRef, error) {
	if order.SourceRunID == nil {
		return nil, nil
	}

	run, err := models.FindUnscopedCanvasRun(tx, *order.SourceRunID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	ref := &factory.AutomationRef{
		AppID:  run.WorkflowID,
		NodeID: run.NodeID,
	}

	if run.NodeID != "" {
		node, err := models.FindCanvasNode(tx, run.WorkflowID, run.NodeID)
		if err == nil {
			ref.NodeName = node.Name
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	canvas, err := models.FindUnscopedCanvasInTransaction(tx, run.WorkflowID)
	if err == nil {
		ref.AppName = canvas.Name
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return ref, nil
}
