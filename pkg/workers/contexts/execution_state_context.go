package contexts

import (
	"time"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionStateContext struct {
	execution *models.WorkflowNodeExecution
	tx        *gorm.DB
}

func NewExecutionStateContext(tx *gorm.DB, execution *models.WorkflowNodeExecution) components.ExecutionStateContext {
	return &ExecutionStateContext{tx: tx, execution: execution}
}

func (s *ExecutionStateContext) Pass(outputs map[string][]any) error {
	events, err := s.execution.PassInTransaction(s.tx, outputs)
	if err != nil {
		return err
	}

	messages.NewWorkflowExecutionFinishedMessage(s.execution.WorkflowID.String(), s.execution).PublishWithDelay(1 * time.Second)
	messages.PublishManyWorkflowEventsWithDelay(s.execution.WorkflowID.String(), events, 1*time.Second)

	return nil
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	err := s.execution.FailInTransaction(s.tx, reason, message)
	messages.NewWorkflowExecutionFinishedMessage(s.execution.WorkflowID.String(), s.execution).PublishWithDelay(1 * time.Second)
	return err
}
