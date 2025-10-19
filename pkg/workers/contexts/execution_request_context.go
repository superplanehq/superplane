package contexts

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
)

type ExecutionRequestContext struct {
	execution *models.WorkflowNodeExecution
}

func NewComponentRequestContext(execution *models.WorkflowNodeExecution) components.RequestContext {
	return &ExecutionRequestContext{execution: execution}
}

func (c *ExecutionRequestContext) ScheduleActionCall(actionName string, parameters map[string]any) error {
	return fmt.Errorf("TODO")
}

func (c *ExecutionRequestContext) SubscribeTo(eventName string, actionName string) error {
	return fmt.Errorf("TODO")
}
