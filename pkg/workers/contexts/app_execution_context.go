package contexts

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
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
	return models.CreateAppMessage(c.tx, c.canvas.ID, c.node.NodeID, payload)
}

func (c *AppExecutionContext) Invoke(app string, node string, payload any) error {
	appContext := NewAppContext(c.tx, c.canvas, c.node)
	targetApp, err := appContext.Get(app)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("target app %s not found", app)
		}
		return err
	}

	targetNode, err := appContext.GetNode(app, node)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("target node %s not found in app %s", node, app)
		}
		return err
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	targetCanvasID := uuid.MustParse(targetApp.ID)
	invocation := models.AppInvocation{
		ID:                uuid.New(),
		CallerAppID:       c.canvas.ID,
		CallerExecutionID: &c.execution.ID,
		TargetCanvasID:    &targetCanvasID,
		TargetNodeID:      targetNode.ID,
		State:             models.AppInvocationStatePending,
		Payload:           datatypes.JSON(payloadBytes),
	}

	return c.tx.Create(&invocation).Error
}
