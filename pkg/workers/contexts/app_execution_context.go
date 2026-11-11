package contexts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type AppExecutionContext struct {
	tx        *gorm.DB
	canvas    *models.Canvas
	node      *models.CanvasNode
	execution *models.CanvasNodeExecution
}

func NewAppExecutionContext(
	tx *gorm.DB,
	canvas *models.Canvas,
	node *models.CanvasNode,
	execution *models.CanvasNodeExecution,
) *AppExecutionContext {
	return &AppExecutionContext{
		tx:        tx,
		canvas:    canvas,
		node:      node,
		execution: execution,
	}
}

func (c *AppExecutionContext) Broadcast(payload any) error {
	var sourceRunID *uuid.UUID
	if c.execution != nil && c.execution.RunID != uuid.Nil {
		sourceRunID = &c.execution.RunID
	}
	return models.CreateAppMessage(c.tx, c.canvas.ID, c.node.NodeID, payload, sourceRunID)
}
