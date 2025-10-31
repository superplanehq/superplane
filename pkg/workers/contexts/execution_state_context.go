package contexts

import (
	log "github.com/sirupsen/logrus"
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

	// Publish RabbitMQ messages for created events
	for _, event := range events {
		err := messages.NewWorkflowEventCreatedMessage(event.WorkflowID.String(), &event).Publish()
		if err != nil {
			log.Errorf("failed to publish workflow event: %v", err)
		}
	}

	return nil
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	return s.execution.FailInTransaction(s.tx, reason, message)
}
