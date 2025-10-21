package contexts

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type NodeRequestContext struct {
	tx   *gorm.DB
	node *models.WorkflowNode
}

func NewNodeRequestContext(tx *gorm.DB, node *models.WorkflowNode) components.RequestContext {
	return &NodeRequestContext{tx: tx, node: node}
}

func (c *NodeRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	if interval < time.Minute {
		return fmt.Errorf("interval must be at least 1 minute")
	}

	runAt := time.Now().Add(interval)
	return c.node.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}, &runAt)
}
