package contexts

import (
	"github.com/superplanehq/superplane/pkg/components"
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
	return s.execution.PassInTransaction(s.tx, outputs)
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	return s.execution.FailInTransaction(s.tx, reason, message)
}
