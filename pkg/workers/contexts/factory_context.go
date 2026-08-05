package contexts

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type FactoryContext struct {
	tx        *gorm.DB
	canvas    *models.Canvas
	execution *models.CanvasNodeExecution
}

func NewFactoryContext(tx *gorm.DB, canvas *models.Canvas, execution *models.CanvasNodeExecution) *FactoryContext {
	return &FactoryContext{
		tx:        tx,
		canvas:    canvas,
		execution: execution,
	}
}

func (c *FactoryContext) CreateWorkOrder(params core.WorkOrderParams) (*core.WorkOrder, error) {
	//
	// Sanity check: do not allow work order to be created
	// when it is already part of a work order execution
	//
	_, err := models.FindWorkOrderExecutionByRunID(c.tx, c.execution.RunID)
	if err == nil {
		return nil, errors.New("cannot create work order while executing another work order")
	}
	if !errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
		return nil, err
	}

	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	factory, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	sourceRunID := c.execution.RunID
	workOrder, err := factory.CreateWorkOrder(c.tx, params.Title, params.Description, nil, []uuid.UUID{}, &sourceRunID)
	if err != nil {
		return nil, err
	}

	return &core.WorkOrder{
		ID:          workOrder.ID.String(),
		Title:       workOrder.Title,
		Description: workOrder.Description,
	}, nil
}
