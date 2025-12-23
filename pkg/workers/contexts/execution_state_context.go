package contexts

import (
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionStateContext struct {
	execution *models.WorkflowNodeExecution
	tx        *gorm.DB
}

func NewExecutionStateContext(tx *gorm.DB, execution *models.WorkflowNodeExecution) *ExecutionStateContext {
	return &ExecutionStateContext{tx: tx, execution: execution}
}

func (s *ExecutionStateContext) IsFinished() bool {
	return s.execution.State == models.WorkflowNodeExecutionStateFinished
}

func (s *ExecutionStateContext) Pass() error {
	_, err := s.execution.PassInTransaction(s.tx, map[string][]any{})
	if err != nil {
		return err
	}

	return nil
}

func (s *ExecutionStateContext) Emit(channel, payloadType string, payloads []any) error {
	outputs := map[string][]any{
		channel: {},
	}

	for _, payload := range payloads {
		outputs[channel] = append(outputs[channel], map[string]any{
			"type":      payloadType,
			"timestamp": time.Now(),
			"data":      payload,
		})
	}

	_, err := s.execution.PassInTransaction(s.tx, outputs)
	if err != nil {
		return err
	}

	return nil
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	err := s.execution.FailInTransaction(s.tx, reason, message)
	return err
}

func (s *ExecutionStateContext) SetKV(key, value string) error {
	return models.CreateWorkflowNodeExecutionKVInTransaction(s.tx, s.execution.WorkflowID, s.execution.NodeID, s.execution.ID, key, value)
}
