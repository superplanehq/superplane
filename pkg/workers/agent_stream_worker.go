package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	agentStreamTimeout   = 15 * time.Minute
	maxConcurrentStreams = 10
	// stuckStreamGrace pads the per-turn timeout before the cleanup loop
	// considers a "streaming" row leaked. A turn that legitimately ran for
	// the full timeout should have been transitioned to idle or failed by
	// then; anything still streaming past 2× is stuck.
	stuckStreamGrace    = 2 * agentStreamTimeout
	stuckCleanupCadence = 5 * time.Minute
)

// AgentStreamWorker is stateless and safe to run as competing consumers.
type AgentStreamWorker struct {
	provider    agents.Provider
	rabbitMQURL string
	slots       chan struct{}
}

func NewAgentStreamWorker(provider agents.Provider, rabbitMQURL string) *AgentStreamWorker {
	return &AgentStreamWorker{
		provider:    provider,
		rabbitMQURL: rabbitMQURL,
		slots:       make(chan struct{}, maxConcurrentStreams),
	}
}

func (w *AgentStreamWorker) Start(ctx context.Context) {
	go w.runStuckSessionCleanup(ctx)

	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"worker":    "agent_stream",
		"route_key": messages.AgentStreamRequestedRoutingKey,
	}))

	for ctx.Err() == nil {
		consumer := tackle.NewConsumer()
		consumer.SetLogger(logger)

		err := consumer.Start(&tackle.Options{
			URL:            w.rabbitMQURL,
			RemoteExchange: messages.CanvasExchange,
			Service:        "superplane.agent-stream-worker",
			RoutingKey:     messages.AgentStreamRequestedRoutingKey,
		}, func(delivery tackle.Delivery) error {
			return w.dispatch(ctx, delivery.Body())
		})

		if err != nil {
			log.WithError(err).Error("agent stream consumer error, reconnecting")
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// dispatch acquires a concurrency slot (back-pressures the consumer when N
// streams are in flight), spawns the worker, and returns so the message can
// be ACKed. A panic in the goroutine is recovered and logged so one bad
// stream can't kill the worker.
func (w *AgentStreamWorker) dispatch(ctx context.Context, body []byte) error {
	select {
	case w.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	go func() {
		defer func() {
			<-w.slots
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("agent stream handler panicked")
			}
		}()
		if err := w.handle(ctx, body); err != nil {
			log.WithError(err).Error("agent stream handler error")
		}
	}()
	return nil
}

func (w *AgentStreamWorker) runStuckSessionCleanup(ctx context.Context) {
	ticker := time.NewTicker(stuckCleanupCadence)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.cleanupTickSafely()
		}
	}
}

// cleanupTickSafely keeps the loop alive across panics by recovering per
// iteration, so a single bad tick doesn't kill the goroutine for the rest
// of the process lifetime.
func (w *AgentStreamWorker) cleanupTickSafely() {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("panic", r).Error("agent stream cleanup panicked")
		}
	}()
	w.cleanupStuckSessions()
}

func (w *AgentStreamWorker) cleanupStuckSessions() {
	cutoff := time.Now().Add(-stuckStreamGrace)
	closed, err := models.FailStuckStreamingSessions(cutoff)
	if err != nil {
		log.WithError(err).Warn("agent stream cleanup: query failed")
		return
	}
	for _, session := range closed {
		if err := messages.PublishAgentSessionEvent(messages.AgentSessionEventMessage{
			SessionID: session.ID.String(),
			Event:     "session_failed",
			Status:    models.AgentSessionStatusFailed,
			Error:     "stream timed out",
		}); err != nil {
			log.WithError(err).WithField("session_id", session.ID).Warn("agent stream cleanup: failed to publish")
		}
	}
}

// Handle is exported only so tests can drive the worker without RabbitMQ.
func (w *AgentStreamWorker) Handle(parentCtx context.Context, body []byte) error {
	return w.handle(parentCtx, body)
}

func (w *AgentStreamWorker) handle(parentCtx context.Context, body []byte) error {
	var req messages.AgentStreamRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.WithError(err).Error("agent stream: invalid request body, dropping")
		return nil
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		log.WithField("session_id", req.SessionID).Warn("agent stream: invalid session id, dropping")
		return nil
	}

	session, err := models.FindAgentSession(sessionID)
	if err != nil {
		log.WithError(err).WithField("session_id", sessionID).Warn("agent stream: session not found, dropping")
		return nil
	}
	if session.Provider != w.provider.Name() {
		// Another replica owns this provider; drop rather than requeue.
		log.WithFields(log.Fields{
			"session_provider": session.Provider,
			"local_provider":   w.provider.Name(),
		}).Warn("agent stream: provider mismatch, dropping")
		return nil
	}

	ctx, cancel := context.WithTimeout(parentCtx, agentStreamTimeout)
	defer cancel()

	publish := func(event messages.AgentSessionEventMessage) {
		event.SessionID = sessionID.String()
		if err := messages.PublishAgentSessionEvent(event); err != nil {
			log.WithError(err).Warn("agent stream: failed to publish event")
		}
	}

	publish(messages.AgentSessionEventMessage{Event: "stream_started", Status: models.AgentSessionStatusStreaming})

	var assistantBuffer strings.Builder
	var streamErr error

	err = w.provider.StreamEvents(ctx, session.ProviderSessionID, func(evt agents.ProviderEvent) error {
		switch evt.Type {
		case agents.ProviderEventAssistantMessage:
			assistantBuffer.WriteString(evt.Text)
			publish(messages.AgentSessionEventMessage{
				Event: "assistant_delta",
				Extra: map[string]any{"text": evt.Text},
			})
		case agents.ProviderEventToolUseStarted:
			// Anthropic's built-in toolset runs tools sequentially and
			// often skips tool_result events, so a fresh tool_use
			// implies any previous in-flight tool finished. Close
			// them before persisting the new row.
			closeOpenTools(sessionID, publish)
			if err := persistAndBroadcast(sessionID, &models.AgentSessionMessage{
				SessionID:       sessionID,
				ProviderEventID: evt.ProviderEventID,
				Role:            models.AgentMessageRoleTool,
				ToolCallID:      evt.ToolCallID,
				ToolName:        evt.ToolName,
				ToolStatus:      models.AgentToolStatusStarted,
				Content:         evt.ToolInput,
			}, "tool_started", publish); err != nil {
				return err
			}
		case agents.ProviderEventToolUseFinished:
			if err := persistAndBroadcast(sessionID, &models.AgentSessionMessage{
				SessionID:       sessionID,
				ProviderEventID: evt.ProviderEventID,
				Role:            models.AgentMessageRoleTool,
				ToolCallID:      evt.ToolCallID,
				ToolName:        evt.ToolName,
				ToolStatus:      models.AgentToolStatusFinished,
			}, "tool_finished", publish); err != nil {
				return err
			}
		case agents.ProviderEventTurnCompleted:
			if assistantBuffer.Len() > 0 {
				if err := persistAndBroadcast(sessionID, &models.AgentSessionMessage{
					SessionID: sessionID,
					Role:      models.AgentMessageRoleAssistant,
					Content:   assistantBuffer.String(),
				}, "assistant_message", publish); err != nil {
					return err
				}
				assistantBuffer.Reset()
			}
			publish(messages.AgentSessionEventMessage{Event: "turn_completed", Status: models.AgentSessionStatusIdle})
		case agents.ProviderEventSessionFailed:
			// Don't publish here — the post-loop block owns
			// session_failed broadcasting so it stays single-source.
			streamErr = fmt.Errorf("provider reported session failed: %s", evt.ErrorMessage)
		}
		return nil
	})

	if streamErr == nil && err != nil && !isContextCancel(err) {
		streamErr = err
	}

	closeOpenTools(sessionID, publish)

	if streamErr != nil {
		_ = models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusFailed)
		publish(messages.AgentSessionEventMessage{
			Event:  "session_failed",
			Status: models.AgentSessionStatusFailed,
			Error:  streamErr.Error(),
		})
		log.WithError(streamErr).WithField("session_id", sessionID).Warn("agent stream: provider stream ended with error")
		return nil
	}

	if err := models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusIdle); err != nil {
		log.WithError(err).Warn("agent stream: failed to mark session idle")
	}
	return nil
}

// closeOpenTools is the safety net for providers that emit tool_use but not
// tool_result (e.g. Anthropic's built-in toolset). Without this, phantom
// "Running" rows would stick around forever.
func closeOpenTools(sessionID uuid.UUID, publish func(messages.AgentSessionEventMessage)) {
	closed, err := models.CloseOpenToolMessages(sessionID)
	if err != nil {
		log.WithError(err).WithField("session_id", sessionID).Warn("agent stream: failed to close open tools")
		return
	}
	for i := range closed {
		publish(messages.AgentSessionEventMessage{
			Event:     "tool_finished",
			MessageID: closed[i].ID.String(),
			Message:   serializeMessage(&closed[i]),
		})
	}
}

// persistAndBroadcast re-reads the row after insert so the websocket payload
// matches the row gRPC ListMessages returns — otherwise the frontend would
// see two distinct objects for the same message.
func persistAndBroadcast(
	sessionID uuid.UUID,
	message *models.AgentSessionMessage,
	eventName string,
	publish func(messages.AgentSessionEventMessage),
) error {
	var stored *models.AgentSessionMessage
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := models.AppendAgentSessionMessageInTransaction(tx, message); err != nil {
			return err
		}

		var fetched models.AgentSessionMessage
		query := tx.Where("session_id = ?", sessionID)
		if message.ProviderEventID != "" {
			query = query.Where("provider_event_id = ?", message.ProviderEventID)
		} else {
			query = query.Where("id = ?", message.ID)
		}
		if err := query.First(&fetched).Error; err != nil {
			return err
		}
		stored = &fetched
		return nil
	})
	if err != nil {
		return fmt.Errorf("persist message: %w", err)
	}

	publish(messages.AgentSessionEventMessage{
		Event:     eventName,
		MessageID: stored.ID.String(),
		Message:   serializeMessage(stored),
	})
	return nil
}

func serializeMessage(m *models.AgentSessionMessage) *messages.AgentMessage {
	return &messages.AgentMessage{
		ID:         m.ID.String(),
		Role:       m.Role,
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
		ToolName:   m.ToolName,
		ToolStatus: m.ToolStatus,
		CreatedAt:  m.CreatedAt,
	}
}

func isContextCancel(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
