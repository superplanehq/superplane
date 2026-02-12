package contexts

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
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

	runAt := time.Now().Add(interval)
	request := models.CanvasNodeRequest{
		WorkflowID: c.node.WorkflowID,
		NodeID:     c.node.NodeID,
		ID:         uuid.New(),
		State:      models.NodeExecutionRequestStatePending,
		Type:       models.NodeRequestTypeInvokeAction,
		Attempts:   0,
		RunAt:      runAt,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: actionName,
				Parameters: parameters,
			},
		}),
	}

	return c.tx.Create(&request).Error
}

func (c *NodeRequestContext) ScheduleActionWithRetry(actionName string, parameters map[string]any, interval time.Duration, maxAttempts int) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	runAt := time.Now().Add(interval)
	request := models.CanvasNodeRequest{
		WorkflowID: c.node.WorkflowID,
		NodeID:     c.node.NodeID,
		ID:         uuid.New(),
		State:      models.NodeExecutionRequestStatePending,
		Type:       models.NodeRequestTypeInvokeAction,
		Attempts:   0,
		RunAt:      runAt,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: actionName,
				Parameters: parameters,
			},
		}),
	}

	if maxAttempts > 1 {
		request.RetryStrategy = datatypes.NewJSONType(models.RetryStrategy{
			Type: models.RetryStrategyTypeConstant,
			Constant: &models.ConstantRetryStrategy{
				MaxAttempts: maxAttempts,
				Delay:       int(interval / time.Second),
			},
		})
	}

	return c.tx.Create(&request).Error
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
