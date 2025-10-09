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

func (s *ExecutionStateContext) Wait() error {
	return s.execution.Wait()
}

func (s *ExecutionStateContext) Finish(outputs map[string][]any) error {
	if err := s.execution.Pass(outputs); err != nil {
		return err
	}

	// Move execution to routing state so router picks it up
	return s.execution.Route()
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	return s.execution.Fail(reason, message)
}
