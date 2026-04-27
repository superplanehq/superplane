package contexts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionStateContext struct {
	execution      *models.CanvasNodeExecution
	tx             *gorm.DB
	maxPayloadSize int
	onNewEvents    func([]models.CanvasEvent)
	action         core.Action

	// channels is resolved lazily on the first Emit call so callers do not
	// have to pre-resolve action.OutputChannels at every construction site.
	channelsResolved bool
	channels         map[string]core.OutputChannel
}

func NewExecutionStateContext(
	tx *gorm.DB,
	action core.Action,
	execution *models.CanvasNodeExecution,
	onNewEvents func([]models.CanvasEvent),
) *ExecutionStateContext {
	return &ExecutionStateContext{
		tx:             tx,
		execution:      execution,
		action:         action,
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

	var (
		newEvents []models.CanvasEvent
		err       error
	)

	if s.isFailureChannel(channel) {
		reason, message := failureDetails(channel, payloadType)
		newEvents, err = s.execution.FailWithOutputsInTransaction(s.tx, outputs, reason, message)
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

func (s *ExecutionStateContext) Fail(reason, message string) error {
	err := s.execution.FailInTransaction(s.tx, reason, message)
	return err
}

func (s *ExecutionStateContext) SetKV(key, value string) error {
	return models.CreateNodeExecutionKVInTransaction(s.tx, s.execution.WorkflowID, s.execution.NodeID, s.execution.ID, key, value)
}

// isFailureChannel reports whether emitting to the given channel should
// finish the execution with a failed result. Derived from the channel's
// Label via core.OutputChannel.IsFailure (a central vocabulary check),
// so actions don't declare failure semantics separately from their
// labels.
func (s *ExecutionStateContext) isFailureChannel(channel string) bool {
	ch, ok := s.resolveChannels()[channel]
	return ok && ch.IsFailure()
}

func (s *ExecutionStateContext) resolveChannels() map[string]core.OutputChannel {
	if s.channelsResolved {
		return s.channels
	}
	s.channelsResolved = true
	s.channels = map[string]core.OutputChannel{}

	if s.action == nil {
		return s.channels
	}

	for _, ch := range s.action.OutputChannels(s.execution.Configuration.Data()) {
		s.channels[ch.Name] = ch
	}
	return s.channels
}

func failureDetails(channel, payloadType string) (string, string) {
	reason := models.CanvasNodeExecutionResultReasonError
	message := fmt.Sprintf("routed to failure channel %q", channel)
	if payloadType != "" {
		message = fmt.Sprintf("routed to failure channel %q (%s)", channel, payloadType)
	}
	return reason, message
}
