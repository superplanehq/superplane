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
	rabbit "github.com/rabbitmq/amqp091-go"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage"
	"gorm.io/gorm"
)

const (
	agentStreamTimeout         = 15 * time.Minute
	maxConcurrentStreams       = 10
	maxLockedStreamReschedules = 10
	agentStreamService         = "superplane.agent-stream-worker"
	streamHeartbeatInterval    = 30 * time.Second
	agentTokenUsagePublishWait = 30 * time.Second
	stuckHeartbeatGrace        = 5 * time.Minute
	// Must stay above agentStreamTimeout so long-but-healthy turns
	// without a heartbeat yet aren't force-failed mid-flight.
	stuckLegacyGrace    = 2 * agentStreamTimeout
	stuckCleanupCadence = 1 * time.Minute
)

var errCustomToolResultsRequired = errors.New("custom tool results required")
var errAgentStreamAlreadyLocked = errors.New("agent stream already in progress")
var errSessionAlreadyReset = errors.New("agent session no longer streaming")

var publishAgentRunFinished = func(session *models.AgentSession, evt agents.ProviderEvent, usageID string) error {
	return messages.NewAgentRunFinishedMessage(
		session.OrganizationID.String(),
		session.ID.String(),
		evt.Model,
		usageID,
		session.ID.String(),
		evt.Usage.InputTokens,
		evt.Usage.OutputTokens,
		evt.Usage.TotalTokens,
		evt.Usage.CacheReadTokens,
		evt.Usage.CacheWriteTokens,
	).Publish()
}

var publishAgentTokenUsageAsync = func(
	ctx context.Context,
	usageService usage.Service,
	session *models.AgentSession,
	evt agents.ProviderEvent,
	usageID string,
) {
	go func() {
		publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), agentTokenUsagePublishWait)
		defer cancel()
		publishPreparedAgentTokenUsage(publishCtx, usageService, session, evt, usageID)
	}()
}

// AgentStreamWorker is stateless and safe to run as competing consumers.
type AgentStreamWorker struct {
	provider           agents.Provider
	customToolExecutor agents.CustomToolExecutor
	usageService       usage.Service
	rabbitMQURL        string
	slots              chan struct{}
}

func NewAgentStreamWorker(provider agents.Provider, rabbitMQURL string, customToolExecutor ...agents.CustomToolExecutor) *AgentStreamWorker {
	return NewAgentStreamWorkerWithUsageService(provider, rabbitMQURL, nil, customToolExecutor...)
}

func NewAgentStreamWorkerWithUsageService(
	provider agents.Provider,
	rabbitMQURL string,
	usageService usage.Service,
	customToolExecutor ...agents.CustomToolExecutor,
) *AgentStreamWorker {
	executor := agents.CustomToolExecutor(unsupportedCustomToolExecutor{})
	if len(customToolExecutor) > 0 && customToolExecutor[0] != nil {
		executor = customToolExecutor[0]
	}
	return &AgentStreamWorker{
		provider:           provider,
		customToolExecutor: executor,
		usageService:       usageService,
		rabbitMQURL:        rabbitMQURL,
		slots:              make(chan struct{}, maxConcurrentStreams),
	}
}

type unsupportedCustomToolExecutor struct{}

func (unsupportedCustomToolExecutor) ExecuteCustomTool(_ context.Context, _ agents.AgentSessionContext, toolUse agents.CustomToolUse) agents.CustomToolResult {
	content, _ := json.Marshal(map[string]string{
		"error": "custom tool executor is not configured",
		"tool":  toolUse.Name,
	})
	return agents.CustomToolResult{
		CustomToolUseID: toolUse.ID,
		Content:         string(content),
		IsError:         true,
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
			Service:        agentStreamService,
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

// dispatch acquires a concurrency slot before ACKing the queue message. If
// another worker already owns the session lock, it durably reschedules the
// message for the retry queue instead of dropping the follow-up turn. The
// heartbeat starts before the slot wait so a backed-up queue can't let
// the cleanup cutoff elapse before the worker claims the session.
func (w *AgentStreamWorker) dispatch(ctx context.Context, body []byte) error {
	stopHeartbeat := startDispatchHeartbeat(ctx, body)

	select {
	case w.slots <- struct{}{}:
	case <-ctx.Done():
		stopHeartbeat()
		return ctx.Err()
	}

	request, err := prepareAgentStreamRequest(ctx, body)
	if err != nil {
		<-w.slots
		stopHeartbeat()
		if errors.Is(err, errAgentStreamAlreadyLocked) {
			return w.rescheduleLockedRequest(ctx, body)
		}
		return err
	}
	if request == nil {
		<-w.slots
		stopHeartbeat()
		return nil
	}

	go func() {
		defer func() {
			stopHeartbeat()
			request.unlock()
			<-w.slots
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("agent stream handler panicked")
			}
		}()
		if err := w.handleLocked(ctx, request.req, request.sessionID); err != nil {
			log.WithError(err).Error("agent stream handler error")
		}
	}()
	return nil
}

// startDispatchHeartbeat returns a no-op stopper when the body is
// unparseable so dispatch's return paths can call stop() unconditionally.
func startDispatchHeartbeat(ctx context.Context, body []byte) func() {
	var req messages.AgentStreamRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return func() {}
	}
	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		return func() {}
	}
	heartbeatCtx, cancel := context.WithCancel(ctx)
	go runStreamHeartbeat(heartbeatCtx, sessionID)
	return cancel
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
	now := time.Now()
	heartbeatCutoff := now.Add(-stuckHeartbeatGrace)
	legacyCutoff := now.Add(-stuckLegacyGrace)
	closed, err := models.FailStuckStreamingSessions(heartbeatCutoff, legacyCutoff)
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
	request, err := prepareAgentStreamRequest(parentCtx, body)
	if err != nil {
		return err
	}
	if request == nil {
		return nil
	}
	defer request.unlock()
	return w.handleLocked(parentCtx, request.req, request.sessionID)
}

type lockedAgentStreamRequest struct {
	req       messages.AgentStreamRequest
	sessionID uuid.UUID
	unlock    func()
}

func prepareAgentStreamRequest(parentCtx context.Context, body []byte) (*lockedAgentStreamRequest, error) {
	var req messages.AgentStreamRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.WithError(err).Error("agent stream: invalid request body, dropping")
		return nil, nil
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		log.WithField("session_id", req.SessionID).Warn("agent stream: invalid session id, dropping")
		return nil, nil
	}

	unlock, locked, err := tryAgentStreamLock(parentCtx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("agent stream: acquire session lock: %w", err)
	}
	if !locked {
		log.WithField("session_id", sessionID).Info("agent stream: stream already in progress, scheduling retry")
		return nil, errAgentStreamAlreadyLocked
	}

	return &lockedAgentStreamRequest{
		req:       req,
		sessionID: sessionID,
		unlock:    unlock,
	}, nil
}

func (w *AgentStreamWorker) rescheduleLockedRequest(ctx context.Context, body []byte) error {
	retryBody, err := lockedStreamRetryBody(body)
	if err != nil {
		return err
	}

	publisher, err := tackle.NewPublisher(w.rabbitMQURL, tackle.PublisherOptions{})
	if err != nil {
		return fmt.Errorf("agent stream: create retry publisher: %w", err)
	}
	defer publisher.Close()

	queueName := (&tackle.Options{
		Service:    agentStreamService,
		RoutingKey: messages.AgentStreamRequestedRoutingKey,
	}).GetDelayQueueName()

	if err := publisher.PublishWithContext(ctx, &tackle.PublishParams{
		Body:       retryBody,
		Headers:    rabbit.Table{},
		Exchange:   "",
		RoutingKey: queueName,
	}); err != nil {
		return fmt.Errorf("agent stream: schedule locked request retry: %w", err)
	}
	return nil
}

func lockedStreamRetryBody(body []byte) ([]byte, error) {
	var req messages.AgentStreamRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("agent stream: decode locked request retry: %w", err)
	}
	if req.LockRetryCount >= maxLockedStreamReschedules {
		return nil, errAgentStreamAlreadyLocked
	}

	req.LockRetryCount++
	retryBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("agent stream: encode locked request retry: %w", err)
	}
	return retryBody, nil
}

func (w *AgentStreamWorker) handleLocked(parentCtx context.Context, req messages.AgentStreamRequest, sessionID uuid.UUID) error {
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
	if session.Status != models.AgentSessionStatusStreaming {
		log.WithFields(log.Fields{
			"session_id": sessionID,
			"status":     session.Status,
		}).Info("agent stream: session is no longer streaming, dropping request")
		return nil
	}

	publish := func(event messages.AgentSessionEventMessage) {
		event.SessionID = sessionID.String()
		if err := messages.PublishAgentSessionEvent(event); err != nil {
			log.WithError(err).Warn("agent stream: failed to publish event")
		}
	}

	for {
		turnStartedAt := session.UpdatedAt
		publish(messages.AgentSessionEventMessage{Event: "stream_started", Status: models.AgentSessionStatusStreaming})

		streamErr := w.streamProviderTurn(parentCtx, session, publish)
		closeOpenTools(sessionID, publish)

		if streamErr != nil {
			// Conditional on turnStartedAt so a stream that errors out
			// after the user already hit Stop (InterruptSession bumps
			// updated_at when it resets to idle) can't flip the row from
			// idle back to failed and spook the UI.
			markedFailed, err := models.UpdateAgentSessionStatusIfUnchanged(sessionID, models.AgentSessionStatusFailed, turnStartedAt)
			if err != nil {
				log.WithError(err).WithField("session_id", sessionID).Warn("agent stream: failed to mark session failed")
				return nil
			}
			if markedFailed {
				publish(messages.AgentSessionEventMessage{
					Event:  "session_failed",
					Status: models.AgentSessionStatusFailed,
					Error:  streamErr.Error(),
				})
				log.WithError(streamErr).WithField("session_id", sessionID).Warn("agent stream: provider stream ended with error")
			} else {
				log.WithError(streamErr).WithField("session_id", sessionID).Info("agent stream: stream errored but session was already reset; dropping failed event")
			}
			return nil
		}

		markedIdle, err := models.UpdateAgentSessionStatusIfUnchanged(sessionID, models.AgentSessionStatusIdle, turnStartedAt)
		if err != nil {
			log.WithError(err).Warn("agent stream: failed to mark session idle")
			return nil
		}
		if markedIdle {
			return nil
		}

		session, err = models.FindAgentSession(sessionID)
		if err != nil {
			log.WithError(err).WithField("session_id", sessionID).Warn("agent stream: session not found after turn, stopping")
			return nil
		}
		if session.Status != models.AgentSessionStatusStreaming {
			log.WithFields(log.Fields{
				"session_id": sessionID,
				"status":     session.Status,
			}).Info("agent stream: session changed after turn, stopping")
			return nil
		}
		log.WithField("session_id", sessionID).Info("agent stream: follow-up request arrived while stream was locked, continuing")
	}
}

func (w *AgentStreamWorker) streamProviderTurn(
	parentCtx context.Context,
	session *models.AgentSession,
	publish func(messages.AgentSessionEventMessage),
) error {
	ctx, cancel := context.WithTimeout(parentCtx, agentStreamTimeout)
	defer cancel()

	var streamErr error
	customTools := newCustomToolTurnState()
	usageRetriever, _ := w.provider.(agents.ProviderSessionUsageRetriever)

	for {
		customTools.clearRequirement()
		err := w.provider.StreamEvents(ctx, session.ProviderSessionID, func(evt agents.ProviderEvent) error {
			return handleProviderEvent(ctx, w.usageService, usageRetriever, session, evt, publish, &streamErr, customTools)
		})

		if errors.Is(err, errCustomToolResultsRequired) {
			err = nil
		}
		if errors.Is(err, errSessionAlreadyReset) {
			return nil
		}
		if streamErr == nil && err != nil && !isContextCancel(err) {
			streamErr = err
		}
		if streamErr != nil || !customTools.resultsRequired {
			return streamErr
		}

		if err := w.executeAndSendCustomToolResults(ctx, session, customTools, publish); err != nil {
			return err
		}
	}
}

func handleProviderEvent(
	ctx context.Context,
	usageService usage.Service,
	usageRetriever agents.ProviderSessionUsageRetriever,
	session *models.AgentSession,
	evt agents.ProviderEvent,
	publish func(messages.AgentSessionEventMessage),
	streamErr *error,
	customTools *customToolTurnState,
) error {
	// Drop late events from a turn the user has already stopped — closes
	// the race between InterruptSession's commit and provider SSE bytes
	// already in flight.
	streaming, err := models.IsAgentSessionStreaming(session.ID)
	if err != nil {
		log.WithError(err).WithField("session_id", session.ID).Warn("agent stream: status check failed; processing event anyway")
	} else if !streaming {
		if evt.Type == agents.ProviderEventTurnCompleted {
			publishAgentTokenUsage(ctx, usageService, usageRetriever, session, evt)
		}
		return errSessionAlreadyReset
	}

	switch evt.Type {
	case agents.ProviderEventAssistantMessage:
		return persistAssistantEvent(session.ID, evt, publish)
	case agents.ProviderEventToolUseStarted:
		return persistToolEvent(session.ID, evt, models.AgentToolStatusStarted, evt.ToolInput, "tool_started", publish)
	case agents.ProviderEventToolUseFinished:
		return persistToolEvent(session.ID, evt, models.AgentToolStatusFinished, "", "tool_finished", publish)
	case agents.ProviderEventCustomToolUseStarted:
		customTools.remember(evt)
		return persistToolEvent(session.ID, evt, models.AgentToolStatusStarted, evt.ToolInput, "tool_started", publish)
	case agents.ProviderEventCustomToolResultsRequired:
		customTools.require(evt.CustomToolEventIDs)
		if err := customTools.resolvePersisted(session.ID); err != nil {
			return err
		}
		if customTools.resultsRequired {
			return errCustomToolResultsRequired
		}
	case agents.ProviderEventTurnCompleted:
		publish(messages.AgentSessionEventMessage{Event: "turn_completed", Status: models.AgentSessionStatusIdle})
		publishAgentTokenUsage(ctx, usageService, usageRetriever, session, evt)
	case agents.ProviderEventOutcomeEvaluationStart:
		publishOutcomeEvaluationStart(evt, publish)
	case agents.ProviderEventOutcomeEvaluation:
		publishOutcomeEvaluationEnd(evt, publish)
	case agents.ProviderEventThreadMessageSent:
		return persistSubagentEvent(session.ID, evt, models.AgentToolStatusStarted, "tool_started", publish)
	case agents.ProviderEventThreadMessageReceived:
		return persistSubagentEvent(session.ID, evt, models.AgentToolStatusFinished, "tool_finished", publish)
	case agents.ProviderEventSessionNotice:
		// Ephemeral notice: no status change, no DB row, stream continues.
		publish(messages.AgentSessionEventMessage{Event: "session_notice", Error: evt.ErrorMessage})
	case agents.ProviderEventSessionFailed:
		// Don't publish here — the post-loop block owns
		// session_failed broadcasting so it stays single-source.
		*streamErr = fmt.Errorf("provider reported session failed: %s", evt.ErrorMessage)
	}
	return nil
}

func publishAgentTokenUsage(
	ctx context.Context,
	usageService usage.Service,
	usageRetriever agents.ProviderSessionUsageRetriever,
	session *models.AgentSession,
	evt agents.ProviderEvent,
) {
	if evt.Usage == nil {
		retrieveAndPublishAgentTokenUsage(ctx, usageService, usageRetriever, session, evt)
		return
	}

	if !evt.Usage.HasUsage() {
		return
	}

	cumulativeUsage := agentSessionTokenUsage(evt.Usage)
	deltaUsage, initialized, err := models.CalculateAgentSessionTokenUsageDelta(session.ID, cumulativeUsage)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"session_id":      session.ID,
			"organization_id": session.OrganizationID,
		}).Warn("agent stream: failed to calculate agent token usage delta")
		return
	}
	if !initialized {
		if err := models.MarkAgentSessionTokenUsageTracked(session.ID, cumulativeUsage); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"session_id":      session.ID,
				"organization_id": session.OrganizationID,
			}).Warn("agent stream: failed to initialize agent token usage watermark")
		}
		return
	}
	if !deltaUsage.HasUsage() {
		return
	}

	if err := models.MarkAgentSessionTokenUsageTracked(session.ID, cumulativeUsage); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"session_id":      session.ID,
			"organization_id": session.OrganizationID,
		}).Warn("agent stream: failed to mark agent token usage tracked")
		return
	}

	publishEvent := evt
	publishEvent.Usage = providerTokenUsage(deltaUsage)
	publishAgentTokenUsageAsync(ctx, usageService, session, publishEvent, agentTokenUsageID(session.ID, cumulativeUsage))
}

func retrieveAndPublishAgentTokenUsage(
	ctx context.Context,
	usageService usage.Service,
	usageRetriever agents.ProviderSessionUsageRetriever,
	session *models.AgentSession,
	evt agents.ProviderEvent,
) {
	if usageRetriever == nil {
		return
	}

	go func() {
		retrieveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), agentTokenUsagePublishWait)
		defer cancel()

		usage, err := usageRetriever.RetrieveSessionUsage(retrieveCtx, session.ProviderSessionID)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"session_id":          session.ID,
				"organization_id":     session.OrganizationID,
				"provider_session_id": session.ProviderSessionID,
			}).Warn("agent stream: failed to retrieve completed session usage")
			return
		}

		evt.Usage = usage
		publishAgentTokenUsage(retrieveCtx, usageService, nil, session, evt)
	}()
}

func publishPreparedAgentTokenUsage(
	ctx context.Context,
	usageService usage.Service,
	session *models.AgentSession,
	evt agents.ProviderEvent,
	usageID string,
) {
	if err := syncAgentTokenUsageOrganization(ctx, usageService, session.OrganizationID.String()); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"session_id":      session.ID,
			"organization_id": session.OrganizationID,
		}).Warn("agent stream: failed to sync organization before publishing agent token usage")
	}

	log.WithFields(log.Fields{
		"session_id":         session.ID,
		"organization_id":    session.OrganizationID,
		"usage_id":           usageID,
		"model":              evt.Model,
		"input_tokens":       evt.Usage.InputTokens,
		"output_tokens":      evt.Usage.OutputTokens,
		"total_tokens":       evt.Usage.TotalTokens,
		"cache_read_tokens":  evt.Usage.CacheReadTokens,
		"cache_write_tokens": evt.Usage.CacheWriteTokens,
	}).Info("agent stream: publishing agent token usage")

	if err := publishAgentRunFinished(session, evt, usageID); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"session_id":      session.ID,
			"organization_id": session.OrganizationID,
			"usage_id":        usageID,
		}).Warn("agent stream: failed to publish agent token usage")
		return
	}
}

func agentTokenUsageID(sessionID uuid.UUID, cumulativeUsage models.AgentSessionTokenUsage) string {
	return fmt.Sprintf(
		"%s:%d:%d:%d:%d:%d",
		sessionID,
		cumulativeUsage.InputTokens,
		cumulativeUsage.OutputTokens,
		cumulativeUsage.CacheReadTokens,
		cumulativeUsage.CacheWriteTokens,
		cumulativeUsage.TotalTokens,
	)
}

func agentSessionTokenUsage(usage *agents.TokenUsage) models.AgentSessionTokenUsage {
	return models.AgentSessionTokenUsage{
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func providerTokenUsage(usage models.AgentSessionTokenUsage) *agents.TokenUsage {
	return &agents.TokenUsage{
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func syncAgentTokenUsageOrganization(ctx context.Context, usageService usage.Service, organizationID string) error {
	if usageService == nil || !usageService.Enabled() {
		return nil
	}

	return usage.SyncOrganization(ctx, usageService, organizationID)
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

	results := make([]agents.CustomToolResult, 0, len(customTools.requiredIDs))
	toolUses := map[string]agents.CustomToolUse{}
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
		toolUses[id] = toolUse

		result := w.customToolExecutor.ExecuteCustomTool(ctx, agentSessionContext(session), toolUse)
		results = append(results, result)
	}

	if err := sender.SendCustomToolResults(ctx, session.ProviderSessionID, results); err != nil {
		return err
	}

	for _, result := range results {
		toolUse := toolUses[result.CustomToolUseID]
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
	db, err := database.OpenDedicatedSQLDB("agent-stream-lock", 1)
	if err != nil {
		return nil, false, err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		_ = db.Close()
		return nil, false, err
	}

	key := agentStreamLockKey(sessionID)
	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked); err != nil {
		_ = conn.Close()
		_ = db.Close()
		return nil, false, err
	}

	if !locked {
		_ = conn.Close()
		_ = db.Close()
		return nil, false, nil
	}

	return func() {
		releaseAgentStreamLock(conn, db, key)
	}, true, nil
}

func releaseAgentStreamLock(conn *sql.Conn, db *sql.DB, key int64) {
	defer db.Close()
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

// runStreamHeartbeat writes once up front so cleanup can't race the first
// tick, then ticks until the caller cancels.
func runStreamHeartbeat(ctx context.Context, sessionID uuid.UUID) {
	if err := models.TouchAgentSessionHeartbeat(sessionID); err != nil {
		log.WithError(err).WithField("session_id", sessionID).Warn("agent stream: initial heartbeat failed")
	}
	ticker := time.NewTicker(streamHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := models.TouchAgentSessionHeartbeat(sessionID); err != nil {
				log.WithError(err).WithField("session_id", sessionID).Warn("agent stream: heartbeat failed")
			}
		}
	}
}

func isContextCancel(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
