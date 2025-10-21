package contexts

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type ExecutionRequestContext struct {
	execution *models.WorkflowNodeExecution
}

func NewExecutionRequestContext(execution *models.WorkflowNodeExecution) components.RequestContext {
	return &ExecutionRequestContext{execution: execution}
}

func (c *ExecutionRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	if interval < time.Minute {
		return fmt.Errorf("interval must be at least 1 minute")
	}

	runAt := time.Now().Add(interval)
	return c.execution.CreateRequest(database.Conn(), models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}, &runAt)
}
