package contexts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Runner terminal emit shape (must stay aligned with pkg/components/runner emitRunnerFinished).
const (
	runnerFinishedPayloadType     = "runner.finished"
	runnerFailedOutputChannelName = "failed"
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
		maxPayloadSize: DefaultMaxPayloadSize,
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

	var newEvents []models.CanvasEvent
	var err error
	if payloadType == runnerFinishedPayloadType && channel == runnerFailedOutputChannelName {
		msg := runnerFailureResultMessage(payloads)
		newEvents, err = s.execution.FinishWithOutputsInTransaction(
			s.tx,
			outputs,
			models.CanvasNodeExecutionResultFailed,
			models.CanvasNodeExecutionResultReasonError,
			msg,
		)
	} else {
		newEvents, err = s.execution.PassInTransaction(s.tx, outputs)
	}
	if err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	return nil
}

func runnerFailureResultMessage(payloads []any) string {
	if len(payloads) == 0 {
		return "failed"
	}
	m, ok := payloads[0].(map[string]any)
	if !ok {
		return "failed"
	}
	for _, key := range []string{"error", "message"} {
		if v, ok := m[key]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s
			}
		}
	}
	return "failed"
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	err := s.execution.FailInTransaction(s.tx, reason, message)
	return err
}

func (s *ExecutionStateContext) SetKV(key, value string) error {
	return models.CreateNodeExecutionKVInTransaction(s.tx, s.execution.WorkflowID, s.execution.NodeID, s.execution.ID, key, value)
}
