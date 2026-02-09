package contexts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionStateContext struct {
	execution      *models.CanvasNodeExecution
	tx             *gorm.DB
	maxPayloadSize int
}

func NewExecutionStateContext(tx *gorm.DB, execution *models.CanvasNodeExecution) *ExecutionStateContext {
	return &ExecutionStateContext{tx: tx, execution: execution, maxPayloadSize: DefaultMaxPayloadSize}
}

func (s *ExecutionStateContext) IsFinished() bool {
	return s.execution.State == models.CanvasNodeExecutionStateFinished
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
		event := map[string]any{
			"type":      payloadType,
			"timestamp": time.Now(),
			"data":      payload,
		}

		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		if len(data) > s.maxPayloadSize {
			return fmt.Errorf("event payload too large: %d bytes (max %d)", len(data), s.maxPayloadSize)
		}

		outputs[channel] = append(outputs[channel], json.RawMessage(data))
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
	return models.CreateNodeExecutionKVInTransaction(s.tx, s.execution.WorkflowID, s.execution.NodeID, s.execution.ID, key, value)
}
