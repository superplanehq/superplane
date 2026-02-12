package contexts

import (
	"errors"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type NodeRequestContext struct {
	tx   *gorm.DB
	node *models.CanvasNode
}

func NewNodeRequestContext(tx *gorm.DB, node *models.CanvasNode) *NodeRequestContext {
	return &NodeRequestContext{tx: tx, node: node}
}

func (c *NodeRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	err := c.completeCurrentRequestForNode()
	if err != nil {
		return err
	}

	spec := models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}

	retryStrategy := &models.RetryStrategy{
		Type: models.RetryStrategyTypeConstant,
		Constant: &models.ConstantRetryStrategy{
			MaxAttempts: 1,
			Delay:       interval,
		},
	}

	return c.node.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, spec, retryStrategy)
}

func (c *NodeRequestContext) ScheduleActionWithRetry(actionName string, parameters map[string]any, interval time.Duration, maxAttempts int) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	spec := models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}

	retryStrategy := &models.RetryStrategy{
		Type: models.RetryStrategyTypeConstant,
		Constant: &models.ConstantRetryStrategy{
			MaxAttempts: maxAttempts,
			Delay:       interval,
		},
	}

	return c.node.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, spec, retryStrategy)
}

func (c *NodeRequestContext) completeCurrentRequestForNode() error {
	request, err := models.FindPendingRequestForNode(c.tx, c.node.WorkflowID, c.node.NodeID)
	if err == nil {
		return request.Pass(c.tx)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
}
