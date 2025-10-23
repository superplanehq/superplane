package contexts

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionRequestContext struct {
	tx        *gorm.DB
	execution *models.WorkflowNodeExecution
}

func NewExecutionRequestContext(tx *gorm.DB, execution *models.WorkflowNodeExecution) components.RequestContext {
	return &ExecutionRequestContext{tx: tx, execution: execution}
}

func (c *ExecutionRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	if interval < 10*time.Second {
		return fmt.Errorf("interval must be bigger than 10s")
	}

	runAt := time.Now().Add(interval)
	return c.execution.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}, &runAt)
}
