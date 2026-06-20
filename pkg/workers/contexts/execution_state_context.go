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
		data, err := marshalStructuredPayload(payloadType, payload)
		if err != nil {
			return err
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
		data, err := marshalStructuredPayload(payloadType, payload)
		if err != nil {
			return err
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

func (s *ExecutionStateContext) EmitSubRuns(channel, payloadType string, payloads []any) error {
	newEvents, err := s.createSubRunRootEvents(channel, payloadType, payloads)
	if err != nil {
		return err
	}

	if _, err := s.execution.PassInTransaction(s.tx, map[string][]any{}); err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	return nil
}

func (s *ExecutionStateContext) EmitSubRunsAndContinue(
	channel,
	payloadType string,
	payloads []any,
) ([]string, error) {
	newEvents, err := s.createSubRunRootEvents(channel, payloadType, payloads)
	if err != nil {
		return nil, err
	}

	if _, err := s.execution.EmitOutputsInTransaction(s.tx, map[string][]any{}); err != nil {
		return nil, err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	rootEventIDs := make([]string, 0, len(newEvents))
	for _, event := range newEvents {
		rootEventIDs = append(rootEventIDs, event.ID.String())
	}

	return rootEventIDs, nil
}

func (s *ExecutionStateContext) createSubRunRootEvents(
	channel,
	payloadType string,
	payloads []any,
) ([]models.CanvasEvent, error) {
	if len(payloads) > config.MaxEmitCount() {
		return nil, fmt.Errorf("cannot emit %d sub-runs (max %d per execution)", len(payloads), config.MaxEmitCount())
	}

	newEvents := make([]models.CanvasEvent, 0, len(payloads))
	for _, payload := range payloads {
		data, err := marshalStructuredPayload(payloadType, payload)
		if err != nil {
			return nil, err
		}

		if len(data) > s.maxPayloadSize {
			return nil, fmt.Errorf("event payload too large: %d bytes (max %d)", len(data), s.maxPayloadSize)
		}

		childRun, err := models.CreateChildCanvasRunInTransaction(s.tx, s.execution.RunID, s.execution.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create child run: %w", err)
		}

		now := time.Now()
		newEvent := models.CanvasEvent{
			WorkflowID: s.execution.WorkflowID,
			NodeID:     s.execution.NodeID,
			Channel:    channel,
			Data:       models.NewJSONValue(json.RawMessage(data)),
			RunID:      childRun.ID,
			State:      models.CanvasEventStatePending,
			CreatedAt:  &now,
		}

		if err := s.tx.Create(&newEvent).Error; err != nil {
			return nil, fmt.Errorf("failed to create sub-run root event: %w", err)
		}

		newEvents = append(newEvents, newEvent)
	}

	return newEvents, nil
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

func marshalStructuredPayload(payloadType string, payload any) ([]byte, error) {
	event := map[string]any{
		"type":      payloadType,
		"timestamp": time.Now(),
		"data":      payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return data, nil
}
