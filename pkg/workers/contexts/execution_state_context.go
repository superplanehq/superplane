package contexts

import (
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
)

type ExecutionStateContext struct {
	execution *models.WorkflowNodeExecution
}

func NewExecutionStateContext(execution *models.WorkflowNodeExecution) components.ExecutionStateContext {
	return &ExecutionStateContext{execution: execution}
}

func (s *ExecutionStateContext) Pass(outputs map[string][]any) error {
	return s.execution.Pass(outputs)
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	return s.execution.Fail(reason, message)
}
