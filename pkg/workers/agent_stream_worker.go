package workers

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
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

var errCustomToolResultsRequired = errors.New("custom tool results required")

// AgentStreamWorker is stateless and safe to run as competing consumers.
type AgentStreamWorker struct {
	provider           agents.Provider
	customToolExecutor agents.CustomToolExecutor
	rabbitMQURL        string
	slots              chan struct{}
}

func NewAgentStreamWorker(provider agents.Provider, rabbitMQURL string, customToolExecutor ...agents.CustomToolExecutor) *AgentStreamWorker {
	var executor agents.CustomToolExecutor
	if len(customToolExecutor) > 0 {
		executor = customToolExecutor[0]
	}
	return &AgentStreamWorker{
		provider:           provider,
		customToolExecutor: executor,
		rabbitMQURL:        rabbitMQURL,
		slots:              make(chan struct{}, maxConcurrentStreams),
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

	unlock, locked, err := tryAgentStreamLock(parentCtx, sessionID)
	if err != nil {
		return fmt.Errorf("agent stream: acquire session lock: %w", err)
	}
	if !locked {
		log.WithField("session_id", sessionID).Info("agent stream: stream already in progress, dropping duplicate request")
		return nil
	}
	defer unlock()

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

	var streamErr error
	customTools := newCustomToolTurnState()

	for {
		customTools.clearRequirement()
		err = w.provider.StreamEvents(ctx, session.ProviderSessionID, func(evt agents.ProviderEvent) error {
			return handleProviderEvent(sessionID, evt, publish, &streamErr, customTools)
		})

		if errors.Is(err, errCustomToolResultsRequired) {
			err = nil
		}
		if streamErr == nil && err != nil && !isContextCancel(err) {
			streamErr = err
		}
		if streamErr != nil || !customTools.resultsRequired {
			break
		}

		if err := w.executeAndSendCustomToolResults(ctx, session, customTools, publish); err != nil {
			streamErr = err
			break
		}
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

func handleProviderEvent(
	sessionID uuid.UUID,
	evt agents.ProviderEvent,
	publish func(messages.AgentSessionEventMessage),
	streamErr *error,
	customTools *customToolTurnState,
) error {
	switch evt.Type {
	case agents.ProviderEventAssistantMessage:
		return persistAssistantEvent(sessionID, evt, publish)
	case agents.ProviderEventToolUseStarted:
		return persistToolEvent(sessionID, evt, models.AgentToolStatusStarted, evt.ToolInput, "tool_started", publish)
	case agents.ProviderEventToolUseFinished:
		return persistToolEvent(sessionID, evt, models.AgentToolStatusFinished, "", "tool_finished", publish)
	case agents.ProviderEventCustomToolUseStarted:
		customTools.remember(evt)
		return persistToolEvent(sessionID, evt, models.AgentToolStatusStarted, evt.ToolInput, "tool_started", publish)
	case agents.ProviderEventCustomToolResultsRequired:
		customTools.require(evt.CustomToolEventIDs)
		if err := customTools.resolvePersisted(sessionID); err != nil {
			return err
		}
		if customTools.resultsRequired {
			return errCustomToolResultsRequired
		}
	case agents.ProviderEventTurnCompleted:
		publish(messages.AgentSessionEventMessage{Event: "turn_completed", Status: models.AgentSessionStatusIdle})
	case agents.ProviderEventOutcomeEvaluationStart:
		publishOutcomeEvaluationStart(evt, publish)
	case agents.ProviderEventOutcomeEvaluation:
		publishOutcomeEvaluationEnd(evt, publish)
	case agents.ProviderEventThreadMessageSent:
		return persistSubagentEvent(sessionID, evt, models.AgentToolStatusStarted, "tool_started", publish)
	case agents.ProviderEventThreadMessageReceived:
		return persistSubagentEvent(sessionID, evt, models.AgentToolStatusFinished, "tool_finished", publish)
	case agents.ProviderEventSessionFailed:
		// Don't publish here — the post-loop block owns
		// session_failed broadcasting so it stays single-source.
		*streamErr = fmt.Errorf("provider reported session failed: %s", evt.ErrorMessage)
	}
	return nil
}

func (w *AgentStreamWorker) executeAndSendCustomToolResults(
	ctx context.Context,
	session *models.AgentSession,
	customTools *customToolTurnState,
	publish func(messages.AgentSessionEventMessage),
) error {
	sender, ok := w.provider.(agents.CustomToolResultSender)
	if !ok {
		return fmt.Errorf("provider does not support custom tool results")
	}
	if w.customToolExecutor == nil {
		return fmt.Errorf("custom tool executor is not configured")
	}

	results := make([]agents.CustomToolResult, 0, len(customTools.requiredIDs))
	for _, id := range customTools.requiredIDs {
		toolUse, ok := customTools.pending[id]
		if !ok {
			results = append(results, agents.CustomToolResult{
				CustomToolUseID: id,
				Content:         "custom tool use event not found",
				IsError:         true,
			})
			continue
		}

		result := w.customToolExecutor.ExecuteCustomTool(ctx, agentSessionContext(session), toolUse)
		results = append(results, result)

		status := models.AgentToolStatusFinished
		if result.IsError {
			status = models.AgentToolStatusFailed
		}
		if err := persistToolEvent(session.ID, agents.ProviderEvent{
			ProviderEventID: result.CustomToolUseID,
			Type:            agents.ProviderEventToolUseFinished,
			ToolName:        toolUse.Name,
			ToolCallID:      result.CustomToolUseID,
		}, status, result.Content, "tool_finished", publish); err != nil {
			return err
		}
	}

	if err := sender.SendCustomToolResults(ctx, session.ProviderSessionID, results); err != nil {
		return err
	}
	customTools.markResolved()
	return nil
}

func agentSessionContext(session *models.AgentSession) agents.AgentSessionContext {
	return agents.AgentSessionContext{
		SessionID:         session.ID.String(),
		ProviderSessionID: session.ProviderSessionID,
		OrganizationID:    session.OrganizationID.String(),
		UserID:            session.UserID.String(),
		CanvasID:          session.CanvasID.String(),
	}
}

type customToolTurnState struct {
	pending         map[string]agents.CustomToolUse
	resolved        map[string]struct{}
	requiredIDs     []string
	resultsRequired bool
}

func newCustomToolTurnState() *customToolTurnState {
	return &customToolTurnState{
		pending:  map[string]agents.CustomToolUse{},
		resolved: map[string]struct{}{},
	}
}

func (s *customToolTurnState) remember(evt agents.ProviderEvent) {
	if evt.CustomToolUse == nil || evt.CustomToolUse.ID == "" {
		return
	}
	s.pending[evt.CustomToolUse.ID] = *evt.CustomToolUse
}

func (s *customToolTurnState) require(ids []string) {
	s.requiredIDs = s.requiredIDs[:0]
	for _, id := range ids {
		if _, ok := s.resolved[id]; ok {
			continue
		}
		s.requiredIDs = append(s.requiredIDs, id)
	}
	s.resultsRequired = len(s.requiredIDs) > 0
}

func (s *customToolTurnState) clearRequirement() {
	s.requiredIDs = nil
	s.resultsRequired = false
}

func (s *customToolTurnState) markResolved() {
	for _, id := range s.requiredIDs {
		s.resolved[id] = struct{}{}
	}
	s.clearRequirement()
}

func (s *customToolTurnState) resolvePersisted(sessionID uuid.UUID) error {
	if len(s.requiredIDs) == 0 {
		return nil
	}

	var resolved []string
	if err := database.Conn().
		Model(&models.AgentSessionMessage{}).
		Where("session_id = ?", sessionID).
		Where("provider_event_id IN ?", s.requiredIDs).
		Where("role = ?", models.AgentMessageRoleTool).
		Where("tool_status IN ?", []string{models.AgentToolStatusFinished, models.AgentToolStatusFailed}).
		Pluck("provider_event_id", &resolved).
		Error; err != nil {
		return err
	}

	for _, id := range resolved {
		s.resolved[id] = struct{}{}
	}
	s.require(s.requiredIDs)
	return nil
}

func tryAgentStreamLock(ctx context.Context, sessionID uuid.UUID) (func(), bool, error) {
	db, err := database.Conn().DB()
	if err != nil {
		return nil, false, err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}

	key := agentStreamLockKey(sessionID)
	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked); err != nil {
		_ = conn.Close()
		return nil, false, err
	}

	if !locked {
		_ = conn.Close()
		return nil, false, nil
	}

	return func() {
		releaseAgentStreamLock(conn, key)
	}, true, nil
}

func releaseAgentStreamLock(conn *sql.Conn, key int64) {
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", key); err != nil {
		log.WithError(err).Warn("agent stream: failed to release session lock")
	}
}

func agentStreamLockKey(sessionID uuid.UUID) int64 {
	h := fnv.New64a()
	h.Write(sessionID[:])
	return int64(binary.BigEndian.Uint64(h.Sum(nil))) //nolint:gosec // wraparound is fine; the key only needs to be deterministic
}

func persistAssistantEvent(
	sessionID uuid.UUID,
	evt agents.ProviderEvent,
	publish func(messages.AgentSessionEventMessage),
) error {
	if evt.Text == "" {
		return nil
	}
	return persistAndBroadcast(sessionID, &models.AgentSessionMessage{
		SessionID:       sessionID,
		ProviderEventID: evt.ProviderEventID,
		Role:            models.AgentMessageRoleAssistant,
		Content:         evt.Text,
	}, "assistant_message", publish)
}

func persistToolEvent(
	sessionID uuid.UUID,
	evt agents.ProviderEvent,
	status string,
	content string,
	eventName string,
	publish func(messages.AgentSessionEventMessage),
) error {
	return persistAndBroadcast(sessionID, &models.AgentSessionMessage{
		SessionID:       sessionID,
		ProviderEventID: evt.ProviderEventID,
		Role:            models.AgentMessageRoleTool,
		ToolCallID:      evt.ToolCallID,
		ToolName:        evt.ToolName,
		ToolStatus:      status,
		Content:         content,
	}, eventName, publish)
}

func publishOutcomeEvaluationStart(evt agents.ProviderEvent, publish func(messages.AgentSessionEventMessage)) {
	if evt.OutcomeResult == nil {
		return
	}
	publish(messages.AgentSessionEventMessage{
		Event: "outcome_evaluation_start",
		Extra: map[string]any{
			"iteration": evt.OutcomeResult.Iteration,
		},
	})
}

func publishOutcomeEvaluationEnd(evt agents.ProviderEvent, publish func(messages.AgentSessionEventMessage)) {
	if evt.OutcomeResult == nil {
		return
	}
	publish(messages.AgentSessionEventMessage{
		Event: "outcome_evaluation_end",
		Extra: map[string]any{
			"iteration":   evt.OutcomeResult.Iteration,
			"result":      evt.OutcomeResult.Result,
			"explanation": evt.OutcomeResult.Explanation,
		},
	})
}

func persistSubagentEvent(
	sessionID uuid.UUID,
	evt agents.ProviderEvent,
	status string,
	eventName string,
	publish func(messages.AgentSessionEventMessage),
) error {
	return persistAndBroadcast(sessionID, &models.AgentSessionMessage{
		SessionID:       sessionID,
		ProviderEventID: evt.ProviderEventID,
		Role:            models.AgentMessageRoleTool,
		ToolName:        "subagent:" + evt.AgentName,
		ToolStatus:      status,
		Content:         evt.Text,
	}, eventName, publish)
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
