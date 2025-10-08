package contexts

import (
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
)

type ExecutionStateContext struct {
	execution *models.WorkflowNodeExecution
	event     *models.WorkflowEvent
}

func NewExecutionStateContext(execution *models.WorkflowNodeExecution, event *models.WorkflowEvent) components.ExecutionStateContext {
	return &ExecutionStateContext{execution: execution, event: event}
}

func (s *ExecutionStateContext) Wait() error {
	return s.execution.Wait()
}

func (s *ExecutionStateContext) Finish(outputs map[string][]any) error {
	if err := s.execution.Pass(outputs); err != nil {
		return err
	}
	return s.event.Route()
}

func (s *ExecutionStateContext) Fail(reason string) error {
	return s.execution.Fail(reason)
}
