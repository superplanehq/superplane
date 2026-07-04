package contexts

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionStateContext struct {
	execution      *models.CanvasNodeExecution
	tx             *gorm.DB
	maxPayloadSize int
	onNewEvents    func([]models.CanvasEvent)
}

func NewExecutionStateContext(
	tx *gorm.DB,
	execution *models.CanvasNodeExecution,
	onNewEvents func([]models.CanvasEvent),
) *ExecutionStateContext {
	return &ExecutionStateContext{
		tx:             tx,
		execution:      execution,
		maxPayloadSize: config.MaxPayloadSize(),
		onNewEvents:    onNewEvents,
	}
}

func (s *ExecutionStateContext) IsFinished() bool {
	return s.execution.State == models.CanvasNodeExecutionStateFinished
}

func (s *ExecutionStateContext) Pass() error {
	newEvents, err := s.execution.PassInTransaction(s.tx, map[string][]any{})
	if err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	return nil
}

func (s *ExecutionStateContext) Emit(channel, payloadType string, payloads []any) error {
	if len(payloads) > config.MaxEmitCount() {
		return fmt.Errorf("cannot emit %d events (max %d per execution)", len(payloads), config.MaxEmitCount())
	}

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

	newEvents, err := s.execution.PassInTransaction(s.tx, outputs)
	if err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	return nil
}

func (s *ExecutionStateContext) EmitAndContinue(channel, payloadType string, payloads []any) error {
	if len(payloads) > config.MaxEmitCount() {
		return fmt.Errorf("cannot emit %d events (max %d per execution)", len(payloads), config.MaxEmitCount())
	}

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

	newEvents, err := s.execution.EmitOutputsInTransaction(s.tx, outputs)
	if err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	return nil
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	if err := s.execution.FailInTransaction(s.tx, reason, message); err != nil {
		return err
	}

	if reason == models.CanvasNodeExecutionResultReasonError {
		DispatchOnError(s.tx, s.execution, s.onNewEvents)
	}

	return nil
}

func (s *ExecutionStateContext) SetKV(key, value string) error {
	return models.CreateNodeExecutionKVInTransaction(s.tx, s.execution.WorkflowID, s.execution.NodeID, s.execution.ID, key, value)
}

func (s *ExecutionStateContext) GetKV(key string) (string, error) {
	value, err := models.FindLatestNodeExecutionKVValueInTransaction(s.tx, s.execution.ID, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", core.ErrExecutionKVNotFound
		}
		return "", err
	}
	return value, nil
}
