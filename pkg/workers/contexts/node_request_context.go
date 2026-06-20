package contexts

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type NodeRequestContext struct {
	tx               *gorm.DB
	node             *models.CanvasNode
	currentRequestID *uuid.UUID
}

func NewNodeRequestContext(tx *gorm.DB, node *models.CanvasNode) *NodeRequestContext {
	return &NodeRequestContext{tx: tx, node: node}
}

func NewCurrentNodeRequestContext(tx *gorm.DB, node *models.CanvasNode, requestID uuid.UUID) *NodeRequestContext {
	return &NodeRequestContext{tx: tx, node: node, currentRequestID: &requestID}
}

func (c *NodeRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	return c.ScheduleActionCallAt(actionName, parameters, time.Now().Add(interval))
}

func (c *NodeRequestContext) ScheduleActionCallAt(actionName string, parameters map[string]any, runAt time.Time) error {
	err := c.completeCurrentRequestForNode()
	if err != nil {
		return err
	}

	return c.node.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}, &runAt)
}

func (c *NodeRequestContext) completeCurrentRequestForNode() error {
	if c.currentRequestID != nil {
		return models.CompletePendingNodeRequestInTransaction(c.tx, *c.currentRequestID)
	}

	request, err := models.FindPendingRequestForNode(c.tx, c.node.WorkflowID, c.node.NodeID)
	if err == nil {
		return request.Complete(c.tx)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
}
