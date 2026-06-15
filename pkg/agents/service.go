package agents

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSessionForbidden = errors.New("agent session is owned by another user")

type Service struct {
	provider         Provider
	auth             authorization.Authorization
	memoryStoreLocks sync.Map
}

func NewService(provider Provider, auth authorization.Authorization) *Service {
	return &Service{
		provider: provider,
		auth:     auth,
	}
}

func (s *Service) ProviderName() string { return s.provider.Name() }

// EnsureSession returns the user's single chat session for the given canvas,
// provisioning it on the upstream provider on first call.
func (s *Service) EnsureSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	if err := s.checkAgentPermission(ctx, userID.String(), organizationID.String()); err != nil {
		return nil, err
	}

	existing, err := findCanvasSession(database.Conn(), organizationID, userID, canvasID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		s.prepareMemoryStoreForExistingSession(ctx, organizationID, userID, canvasID)
		return existing, nil
	}
	return s.provisionSession(ctx, organizationID, userID, canvasID)
}

func (s *Service) prepareMemoryStoreForExistingSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) {
	// Anthropic memory stores attach only when a provider session is created.
	// For existing sessions, prepare the mapping for the next recovery without
	// replacing the provider session and losing its current conversation state.
	if _, err := s.memoryStoresForScope(ctx, organizationID, userID, canvasID); err != nil {
		log.WithError(err).
			WithField("provider", s.provider.Name()).
			WithField("organization_id", organizationID).
			WithField("user_id", userID).
			WithField("canvas_id", canvasID).
			Warn("failed to prepare agent memory store for existing session")
	}
}

func (s *Service) provisionSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	session, err := s.existingSessionForProvision(organizationID, userID, canvasID)
	if err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
	}
	if session != nil {
		s.prepareMemoryStoreForExistingSession(ctx, organizationID, userID, canvasID)
		return session, nil
	}

	memoryStores, err := s.memoryStoresForScope(ctx, organizationID, userID, canvasID)
	if err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", ensureSessionLockKey(organizationID, userID, canvasID)).Error; err != nil {
			return err
		}

		found, err := findCanvasSession(tx, organizationID, userID, canvasID)
		if err != nil {
			return err
		}
		if found != nil {
			session = found
			return nil
		}

		title := sessionTitle(organizationID, canvasID)
		upstream, err := s.provider.CreateSession(ctx, CreateSessionOptions{
			Title:        title,
			MemoryStores: memoryStores,
		})
		if err != nil {
			return fmt.Errorf("create provider session: %w", err)
		}

		session = &models.AgentSession{
			OrganizationID:    organizationID,
			UserID:            userID,
			CanvasID:          canvasID,
			Provider:          s.provider.Name(),
			ProviderSessionID: upstream.ProviderSessionID,
			Status:            models.AgentSessionStatusIdle,
		}
		return models.CreateAgentSessionInTransaction(tx, session)
	})
	if err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
	}
	return session, nil
}

func (s *Service) existingSessionForProvision(organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	var session *models.AgentSession
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", ensureSessionLockKey(organizationID, userID, canvasID)).Error; err != nil {
			return err
		}

		found, err := findCanvasSession(tx, organizationID, userID, canvasID)
		if err != nil {
			return err
		}
		session = found
		return nil
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) GetSession(organizationID, userID, sessionID uuid.UUID) (*models.AgentSession, error) {
	return models.FindAgentSessionForUser(organizationID, userID, sessionID)
}

func (s *Service) ListMessages(sessionID, beforeID uuid.UUID, limit int) ([]models.AgentSessionMessage, error) {
	if beforeID == uuid.Nil {
		return models.ListAgentSessionMessagesPage(sessionID, nil, limit)
	}
	cursor, err := findCursorMessage(sessionID, beforeID)
	if err != nil {
		return nil, err
	}
	if cursor == nil {
		return nil, nil
	}
	return models.ListAgentSessionMessagesPage(sessionID, cursor, limit)
}

// InterruptSession resets local state first and treats the provider call
// as best-effort: the worker checks status per event, so flipping to idle
// before the provider HTTP roundtrip lets late SSE events get dropped
// during the network wait instead of after.
func (s *Service) InterruptSession(ctx context.Context, organizationID, userID, sessionID uuid.UUID) error {
	session, err := s.GetSession(organizationID, userID, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	closeOpenToolsForSession(sessionID)

	if err := models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusIdle); err != nil {
		return fmt.Errorf("mark session idle: %w", err)
	}

	if err := messages.PublishAgentSessionEvent(messages.AgentSessionEventMessage{
		SessionID: sessionID.String(),
		Event:     "turn_completed",
		Status:    models.AgentSessionStatusIdle,
	}); err != nil {
		log.WithError(err).WithField("session_id", sessionID).Warn("interrupt: failed to publish status event")
	}

	// Best-effort: any provider error still leaves the row at idle so the
	// stop button can't leave the UI gated; SendMessage's recovery paths
	// reconcile on the next turn.
	if err := s.provider.InterruptSession(ctx, session.ProviderSessionID); err != nil {
		if errors.Is(err, ErrProviderSessionUnavailable) {
			log.WithField("session_id", sessionID).Info("interrupt: provider session unavailable, local reset already done")
		} else {
			log.WithError(err).WithField("session_id", sessionID).Warn("interrupt: provider returned error, local reset already done")
		}
	}
	return nil
}

// closeOpenToolsForSession mirrors agent_stream_worker.closeOpenTools;
// duplicated to avoid a workers → agents → workers import cycle.
func closeOpenToolsForSession(sessionID uuid.UUID) {
	closed, err := models.CloseOpenToolMessages(sessionID)
	if err != nil {
		log.WithError(err).WithField("session_id", sessionID).Warn("interrupt: failed to close open tools")
		return
	}
	for i := range closed {
		row := &closed[i]
		if err := messages.PublishAgentSessionEvent(messages.AgentSessionEventMessage{
			SessionID: sessionID.String(),
			Event:     "tool_finished",
			MessageID: row.ID.String(),
			Message: &messages.AgentMessage{
				ID:         row.ID.String(),
				Role:       row.Role,
				Content:    row.Content,
				ToolCallID: row.ToolCallID,
				ToolName:   row.ToolName,
				ToolStatus: row.ToolStatus,
				CreatedAt:  row.CreatedAt,
			},
		}); err != nil {
			log.WithError(err).WithField("session_id", sessionID).Warn("interrupt: failed to publish tool_finished")
		}
	}
}

func (s *Service) DefineOutcome(ctx context.Context, organizationID, userID, sessionID uuid.UUID, description, rubric string, maxIterations int) error {
	session, err := s.GetSession(organizationID, userID, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if session.Status == models.AgentSessionStatusStreaming {
		return s.handleBusySession(sessionID, organizationID, userID)
	}

	preamble := s.buildPreamble(session, ModeBuilder)

	if err := s.provider.DefineOutcome(ctx, session.ProviderSessionID, DefineOutcomeOptions{
		Description:     description,
		Rubric:          rubric,
		MaxIterations:   maxIterations,
		ContextPreamble: preamble,
	}); err != nil {
		if errors.Is(err, ErrSessionBusy) {
			return s.handleBusySession(sessionID, organizationID, userID)
		}
		if errors.Is(err, ErrProviderSessionUnavailable) {
			recovered, recoverErr := s.recoverProviderSession(ctx, session)
			if recoverErr != nil {
				if errors.Is(recoverErr, ErrSessionBusy) {
					return s.handleBusySession(sessionID, organizationID, userID)
				}
				return recoverErr
			}
			if err := s.provider.DefineOutcome(ctx, recovered.ProviderSessionID, DefineOutcomeOptions{
				Description:     description,
				Rubric:          rubric,
				MaxIterations:   maxIterations,
				ContextPreamble: preamble,
			}); err != nil {
				if errors.Is(err, ErrSessionBusy) {
					return s.handleBusySession(sessionID, organizationID, userID)
				}
				return fmt.Errorf("define outcome after provider session recovery: %w", err)
			}
		} else {
			return fmt.Errorf("define outcome: %w", err)
		}
	}

	// Mark session as streaming and start the stream worker to pick up events
	if err := models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusStreaming); err != nil {
		log.WithError(err).Warn("failed to mark agent session as streaming")
	}
	if err := s.enqueueStream(sessionID, organizationID, userID); err != nil {
		log.WithError(err).Warn("failed to enqueue stream after define outcome")
	}
	return nil
}

func (s *Service) SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string, mode ...string) (*models.AgentSessionMessage, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	session, err := models.FindAgentSessionForUser(organizationID, userID, sessionID)
	if err != nil {
		return nil, err
	}

	agentMode := ModeOperator
	if len(mode) > 0 {
		agentMode = NormalizeMode(mode[0])
	}

	preamble := s.buildPreamble(session, agentMode)

	if err := s.provider.SendMessage(ctx, session.ProviderSessionID, content, SendMessageOptions{ContextPreamble: preamble}); err != nil {
		if errors.Is(err, ErrSessionBusy) {
			return nil, s.handleBusySession(sessionID, organizationID, userID)
		}
		if errors.Is(err, ErrProviderSessionUnavailable) {
			recovered, recoverErr := s.recoverProviderSession(ctx, session)
			if recoverErr != nil {
				if errors.Is(recoverErr, ErrSessionBusy) {
					return nil, s.handleBusySession(sessionID, organizationID, userID)
				}
				return nil, recoverErr
			}
			if err := s.provider.SendMessage(ctx, recovered.ProviderSessionID, content, SendMessageOptions{ContextPreamble: preamble}); err != nil {
				if errors.Is(err, ErrSessionBusy) {
					return nil, s.handleBusySession(sessionID, organizationID, userID)
				}
				return nil, fmt.Errorf("send message after provider session recovery: %w", err)
			}
		} else {
			_ = models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusFailed)
			return nil, fmt.Errorf("forward to provider: %w", err)
		}
	}

	messageRole := models.AgentMessageRoleUser
	if strings.HasPrefix(content, "@@system: ") {
		messageRole = models.AgentMessageRoleSystem
	}
	persisted := &models.AgentSessionMessage{
		SessionID: sessionID,
		Role:      messageRole,
		Content:   content,
	}
	if err := models.AppendAgentSessionMessage(persisted); err != nil {
		return nil, fmt.Errorf("persist user message: %w", err)
	}

	if err := models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusStreaming); err != nil {
		log.WithError(err).Warn("failed to mark agent session as streaming")
	}

	if err := s.enqueueStream(sessionID, organizationID, userID); err != nil {
		return nil, err
	}
	return persisted, nil
}

func (s *Service) handleBusySession(sessionID, organizationID, userID uuid.UUID) error {
	if err := s.enqueueStreamAfterBusySession(sessionID, organizationID, userID); err != nil {
		return err
	}
	return ErrSessionBusy
}

func (s *Service) enqueueStreamAfterBusySession(sessionID, organizationID, userID uuid.UUID) error {
	if err := models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusStreaming); err != nil {
		return fmt.Errorf("mark busy agent session as streaming: %w", err)
	}
	return s.enqueueStream(sessionID, organizationID, userID)
}

func (s *Service) recoverProviderSession(ctx context.Context, stale *models.AgentSession) (*models.AgentSession, error) {
	target, err := s.providerSessionRecoveryTarget(stale)
	if err != nil {
		return nil, fmt.Errorf("recover provider session: %w", err)
	}
	if target.recovered != nil {
		return target.recovered, nil
	}

	memoryStores, err := s.memoryStoresForScope(ctx, target.organizationID, target.userID, target.canvasID)
	if err != nil {
		return nil, fmt.Errorf("recover provider session: load memory store: %w", err)
	}

	upstream, err := s.provider.CreateSession(ctx, CreateSessionOptions{
		Title:        sessionTitle(target.organizationID, target.canvasID),
		MemoryStores: memoryStores,
	})
	if err != nil {
		return nil, fmt.Errorf("recover provider session: create provider session: %w", err)
	}

	recovered, err := s.installRecoveredProviderSession(stale, upstream.ProviderSessionID)
	if err != nil {
		s.cleanupProviderSession(ctx, upstream.ProviderSessionID)
		return nil, fmt.Errorf("recover provider session: %w", err)
	}
	if recovered.ProviderSessionID != upstream.ProviderSessionID {
		s.cleanupProviderSession(ctx, upstream.ProviderSessionID)
	}
	return recovered, nil
}

type providerSessionRecoveryTarget struct {
	organizationID uuid.UUID
	userID         uuid.UUID
	canvasID       uuid.UUID
	recovered      *models.AgentSession
}

func (s *Service) providerSessionRecoveryTarget(stale *models.AgentSession) (*providerSessionRecoveryTarget, error) {
	target := &providerSessionRecoveryTarget{}
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockAgentSessionInTransaction(tx, stale.ID)
		if err != nil {
			return err
		}
		if locked.Status == models.AgentSessionStatusStreaming {
			return ErrSessionBusy
		}
		if locked.ProviderSessionID != stale.ProviderSessionID {
			target.recovered = locked
			return nil
		}

		target.organizationID = locked.OrganizationID
		target.userID = locked.UserID
		target.canvasID = locked.CanvasID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return target, nil
}

func (s *Service) installRecoveredProviderSession(stale *models.AgentSession, providerSessionID string) (*models.AgentSession, error) {
	var recovered *models.AgentSession
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockAgentSessionInTransaction(tx, stale.ID)
		if err != nil {
			return err
		}
		if locked.Status == models.AgentSessionStatusStreaming {
			return ErrSessionBusy
		}
		if locked.ProviderSessionID != stale.ProviderSessionID {
			recovered = locked
			return nil
		}

		if err := models.UpdateAgentSessionProviderSessionInTransaction(
			tx,
			locked.ID,
			providerSessionID,
			models.AgentSessionStatusIdle,
		); err != nil {
			return err
		}
		locked.ProviderSessionID = providerSessionID
		locked.Status = models.AgentSessionStatusIdle
		recovered = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	return recovered, nil
}

const (
	agentMemoryStoreDescription  = "Durable SuperPlane agent memory for one user and one app. Store user preferences, app-specific conventions, the latest private draft version ID, useful lessons, and prior mistakes. Do not store secret values, credentials, API tokens, raw prompts, or large app YAML snapshots."
	agentMemoryStoreInstructions = "Use this memory only for durable lessons, user preferences, app-specific conventions, the current private draft/version ID returned by superplane_app for this app, and prior mistakes that will help future SuperPlane agent sessions for this same user and app. Treat remembered draft IDs as hints; latest session context and superplane_app responses are authoritative. Never store secret values, credentials, API tokens, raw user prompts, or large app YAML snapshots."
)

func (s *Service) memoryStoresForScope(
	ctx context.Context,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
) ([]MemoryStoreResource, error) {
	lock := s.memoryStoreLock(organizationID, userID, canvasID)
	lock.Lock()
	defer lock.Unlock()

	plan, err := s.memoryStorePlanForScope(organizationID, userID, canvasID)
	if err != nil {
		return nil, err
	}
	if plan.store != nil {
		return []MemoryStoreResource{memoryStoreResource(plan.store)}, nil
	}
	if !plan.shouldCreate {
		return nil, nil
	}

	created, ok := s.createProviderMemoryStore(ctx, organizationID, userID, canvasID)
	if !ok {
		return nil, nil
	}

	store, err := s.saveMemoryStoreMapping(organizationID, userID, canvasID, created.ProviderMemoryStoreID)
	if err != nil {
		s.cleanupProviderMemoryStore(ctx, created.ProviderMemoryStoreID)
		log.WithError(err).
			WithField("provider", s.provider.Name()).
			WithField("organization_id", organizationID).
			WithField("user_id", userID).
			WithField("canvas_id", canvasID).
			WithField("provider_memory_store_id", created.ProviderMemoryStoreID).
			Warn("failed to save agent memory store mapping; continuing without memory")
		return nil, nil
	}
	if store == nil {
		return nil, nil
	}

	return []MemoryStoreResource{memoryStoreResource(store)}, nil
}

type memoryStorePlan struct {
	store        *models.AgentMemoryStore
	shouldCreate bool
}

func (s *Service) memoryStorePlanForScope(organizationID, userID, canvasID uuid.UUID) (*memoryStorePlan, error) {
	plan := &memoryStorePlan{}
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		enabled := s.agentMemoryEnabledInTransaction(tx, organizationID)
		if !enabled {
			return nil
		}

		if _, ok := s.provider.(MemoryStoreCreator); !ok {
			log.WithField("provider", s.provider.Name()).Warn("agent memory enabled but provider does not support memory stores")
			return nil
		}

		store, err := models.FindAgentMemoryStoreByScopeInTransaction(tx, organizationID, userID, canvasID, s.provider.Name())
		if err == nil {
			plan.store = store
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find agent memory store: %w", err)
		}

		plan.shouldCreate = true
		return nil
	})
	return plan, err
}

func (s *Service) createProviderMemoryStore(
	ctx context.Context,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
) (*CreateMemoryStoreResult, bool) {
	creator, ok := s.provider.(MemoryStoreCreator)
	if !ok {
		return nil, false
	}
	created, err := creator.CreateMemoryStore(ctx, CreateMemoryStoreOptions{
		Name:        agentMemoryStoreName(userID, canvasID),
		Description: agentMemoryStoreDescription,
	})
	if err != nil {
		log.WithError(err).
			WithField("provider", s.provider.Name()).
			WithField("organization_id", organizationID).
			WithField("user_id", userID).
			WithField("canvas_id", canvasID).
			Warn("failed to create agent memory store; continuing without memory")
		return nil, false
	}
	if created.ProviderMemoryStoreID == "" {
		log.WithField("provider", s.provider.Name()).Warn("provider returned empty agent memory store id; continuing without memory")
		return nil, false
	}

	return created, true
}

func (s *Service) saveMemoryStoreMapping(
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
	providerMemoryStoreID string,
) (*models.AgentMemoryStore, error) {
	var saved *models.AgentMemoryStore
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		store, err := models.FindAgentMemoryStoreByScopeInTransaction(tx, organizationID, userID, canvasID, s.provider.Name())
		if err == nil {
			saved = store
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("find agent memory store: %w", err)
		}

		store = &models.AgentMemoryStore{
			OrganizationID:        organizationID,
			UserID:                userID,
			CanvasID:              canvasID,
			Provider:              s.provider.Name(),
			ProviderMemoryStoreID: providerMemoryStoreID,
			Name:                  agentMemoryStoreName(userID, canvasID),
			Description:           agentMemoryStoreDescription,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "organization_id"},
				{Name: "user_id"},
				{Name: "canvas_id"},
				{Name: "provider"},
			},
			DoNothing: true,
		}).Create(store).Error; err != nil {
			return fmt.Errorf("create agent memory store mapping: %w", err)
		}

		saved, err = models.FindAgentMemoryStoreByScopeInTransaction(tx, organizationID, userID, canvasID, s.provider.Name())
		if err != nil {
			return fmt.Errorf("find saved agent memory store mapping: %w", err)
		}
		log.WithField("provider", s.provider.Name()).
			WithField("organization_id", organizationID).
			WithField("user_id", userID).
			WithField("canvas_id", canvasID).
			WithField("provider_memory_store_id", providerMemoryStoreID).
			Info("created agent memory store")
		return nil
	})
	return saved, err
}

func (s *Service) cleanupProviderMemoryStore(ctx context.Context, providerMemoryStoreID string) {
	cleaner, ok := s.provider.(MemoryStoreCleaner)
	if !ok {
		log.WithField("provider", s.provider.Name()).
			WithField("provider_memory_store_id", providerMemoryStoreID).
			Warn("provider memory store was created but cannot be cleaned up after mapping failure")
		return
	}
	if err := cleaner.DeleteMemoryStore(ctx, providerMemoryStoreID); err != nil {
		log.WithError(err).
			WithField("provider", s.provider.Name()).
			WithField("provider_memory_store_id", providerMemoryStoreID).
			Warn("failed to clean up provider memory store after mapping failure")
	}
}

func (s *Service) agentMemoryEnabledInTransaction(tx *gorm.DB, organizationID uuid.UUID) bool {
	if features.IsReleased(features.FeatureClaudeManagedAgentMemory) {
		return true
	}

	organization, err := models.FindOrganizationByIDInTransaction(tx, organizationID.String())
	if err != nil {
		log.WithError(err).WithField("organization_id", organizationID).Warn("failed to check agent memory feature; continuing without memory")
		return false
	}
	return organization.HasExperimentalFeature(features.FeatureClaudeManagedAgentMemory)
}

func (s *Service) memoryStoreLock(organizationID, userID, canvasID uuid.UUID) *sync.Mutex {
	key := agentMemoryStoreLockKey(organizationID, userID, canvasID, s.provider.Name())
	lock, _ := s.memoryStoreLocks.LoadOrStore(key, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func agentMemoryStoreName(userID uuid.UUID, canvasID uuid.UUID) string {
	return fmt.Sprintf("SuperPlane canvas memory %s user %s", shortUUID(canvasID), shortUUID(userID))
}

func shortUUID(id uuid.UUID) string {
	value := id.String()
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func memoryStoreResource(store *models.AgentMemoryStore) MemoryStoreResource {
	return MemoryStoreResource{
		MemoryStoreID: store.ProviderMemoryStoreID,
		Access:        MemoryStoreAccessReadWrite,
		Instructions:  agentMemoryStoreInstructions,
	}
}

func (s *Service) cleanupProviderSession(ctx context.Context, providerSessionID string) {
	cleaner, ok := s.provider.(ProviderSessionCleaner)
	if !ok {
		return
	}
	if err := cleaner.DeleteSession(ctx, providerSessionID); err != nil {
		log.WithError(err).WithField("provider_session_id", providerSessionID).Warn("failed to clean up unused recovered provider session")
	}
}

func (s *Service) buildPreamble(session *models.AgentSession, mode Mode) string {
	base := fmt.Sprintf(
		preambleTemplate,
		session.CanvasID.String(),
		session.OrganizationID.String(),
		session.CanvasID.String(),
		session.CanvasID.String(),
	)
	canvasSnapshot := buildCanvasSnapshot(session)
	draftStatus := getDraftStatus(session.CanvasID)
	return base + "\n\n" + canvasSnapshot + "\n\n" + modeInstructions(mode) + "\n\n" + draftStatus
}

func (s *Service) enqueueStream(sessionID, organizationID, userID uuid.UUID) error {
	err := messages.PublishAgentStreamRequested(messages.AgentStreamRequest{
		SessionID:      sessionID.String(),
		OrganizationID: organizationID.String(),
		UserID:         userID.String(),
	})
	if err == nil {
		return nil
	}
	_ = models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusFailed)
	_ = messages.PublishAgentSessionEvent(messages.AgentSessionEventMessage{
		SessionID: sessionID.String(),
		Event:     "session_failed",
		Status:    models.AgentSessionStatusFailed,
		Error:     "failed to enqueue stream",
	})
	log.WithError(err).Error("failed to enqueue agent stream request")
	return fmt.Errorf("enqueue stream: %w", err)
}

// checkAgentPermission enforces the org-level baseline. Per-canvas access is
// gated by the gRPC handler's models.FindCanvas(orgID, canvasID).
func (s *Service) checkAgentPermission(ctx context.Context, userID, organizationID string) error {
	checks := []struct{ resource, action string }{
		{"agents", "create"},
		{"canvases", "read"},
	}
	for _, c := range checks {
		allowed, err := s.auth.CheckOrganizationPermission(ctx, userID, organizationID, c.resource, c.action)
		if err != nil {
			return fmt.Errorf("resolve %s:%s permission: %w", c.resource, c.action, err)
		}
		if !allowed {
			return ErrSessionForbidden
		}
	}
	return nil
}

func findCanvasSession(tx *gorm.DB, orgID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	session, err := models.FindAgentSessionByCanvasInTransaction(tx, orgID, userID, canvasID)
	if err == nil {
		return session, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("load session: %w", err)
}

func findCursorMessage(sessionID, beforeID uuid.UUID) (*models.AgentSessionMessage, error) {
	var anchor models.AgentSessionMessage
	err := database.Conn().
		Where("session_id = ?", sessionID).
		Where("id = ?", beforeID).
		First(&anchor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &anchor, nil
}

func ensureSessionLockKey(organizationID, userID, canvasID uuid.UUID) int64 {
	h := fnv.New64a()
	h.Write(organizationID[:])
	h.Write(userID[:])
	h.Write(canvasID[:])
	return int64(binary.BigEndian.Uint64(h.Sum(nil))) //nolint:gosec // wraparound is fine; we just need a deterministic key
}

func agentMemoryStoreLockKey(organizationID, userID, canvasID uuid.UUID, provider string) int64 {
	h := fnv.New64a()
	h.Write([]byte("agent-memory-store"))
	h.Write(organizationID[:])
	h.Write(userID[:])
	h.Write(canvasID[:])
	h.Write([]byte(provider))
	return int64(binary.BigEndian.Uint64(h.Sum(nil))) //nolint:gosec // wraparound is fine; we just need a deterministic key
}

func getDraftStatus(canvasID uuid.UUID) string {
	drafts, err := models.ListDraftCanvasVersions(canvasID)
	if err != nil {
		log.WithError(err).Warn("failed to list draft canvas versions")
		return "[Draft Status]\nUnable to determine draft status."
	}

	if len(drafts) > 0 {
		result := "[Draft Status]\n"
		for _, draft := range drafts {
			result += fmt.Sprintf(
				"- Existing draft: version %s (created %s)\n",
				draft.ID.String(),
				draftCreatedAt(draft),
			)
		}
		result += "These drafts may belong to other sessions or users. To make changes, use 'superplane_app' action 'update_draft'; it automatically targets your own private draft, creating one from the live version if needed. Do not assume an unrelated draft is yours.\n"
		return result
	}

	latestPublished, err := models.FindLatestPublishedCanvasVersion(canvasID)
	if err != nil {
		return noActiveDraftStatus
	}
	if !wasRecentlyPublished(latestPublished.PublishedAt) {
		return noActiveDraftStatus
	}

	return fmt.Sprintf(
		"[Draft Status]\nNo active drafts. The last draft was published as version %s at %s. Your changes are live.",
		latestPublished.ID.String(),
		latestPublished.PublishedAt.UTC().Format(time.RFC3339),
	)
}

func buildCanvasSnapshot(session *models.AgentSession) string {
	canvas, err := models.FindCanvas(session.OrganizationID, session.CanvasID)
	if err != nil {
		log.WithError(err).Warn("failed to load canvas for agent snapshot")
		return "[Canvas Snapshot]\nUnable to load current canvas snapshot."
	}

	var builder strings.Builder
	builder.WriteString("[Canvas Snapshot]\n")
	builder.WriteString(fmt.Sprintf("canvas_id: %s\n", canvas.ID.String()))
	builder.WriteString(fmt.Sprintf("name: %s\n", canvas.Name))

	if canvas.LiveVersionID != nil {
		builder.WriteString(fmt.Sprintf("live_version_id: %s\n", canvas.LiveVersionID.String()))
	}

	draft, draftErr := ownedDraftVersion(session.CanvasID, session.UserID)
	if draftErr != nil {
		log.WithError(draftErr).Warn("failed to load owned draft for agent snapshot")
	}

	snapshotSource, snapshotAvailable := appendDraftSnapshotStatus(&builder, draft, draftErr)
	if !snapshotAvailable {
		return strings.TrimRight(builder.String(), "\n")
	}

	version, err := selectedVersion(canvas, draft, snapshotSource)
	if err != nil {
		log.WithError(err).Warn("failed to load canvas version for agent snapshot")
		builder.WriteString("snapshot_source: unavailable\n")
		builder.WriteString("nodes: unavailable\n")
		return strings.TrimRight(builder.String(), "\n")
	}

	if version == nil {
		builder.WriteString(fmt.Sprintf("snapshot_source: %s\n", snapshotSource))
		builder.WriteString("nodes: unavailable\n")
		return strings.TrimRight(builder.String(), "\n")
	}

	builder.WriteString(fmt.Sprintf("snapshot_source: %s\n", snapshotSource))
	builder.WriteString(fmt.Sprintf("node_count: %d\n", len(version.Nodes)))
	builder.WriteString(fmt.Sprintf("edge_count: %d\n", len(version.Edges)))

	nodes := summarizeNodes(version.Nodes, 12)
	if len(nodes) == 0 {
		builder.WriteString("node_summaries: []\n")
		return strings.TrimRight(builder.String(), "\n")
	}

	builder.WriteString("node_summaries:\n")
	for _, node := range nodes {
		component := node.Component
		if component == "" {
			component = "unknown"
		}
		name := node.Name
		if name == "" {
			name = node.ID
		}
		line := fmt.Sprintf("  - id=%s name=%q type=%s component=%s", node.ID, name, node.Type, component)
		if node.Issue != "" {
			line += fmt.Sprintf(" issue=%q", node.Issue)
		}
		builder.WriteString(line + "\n")
	}

	if len(version.Nodes) > len(nodes) {
		builder.WriteString(fmt.Sprintf("  - ... %d more nodes omitted\n", len(version.Nodes)-len(nodes)))
	}

	return strings.TrimRight(builder.String(), "\n")
}

func appendDraftSnapshotStatus(builder *strings.Builder, draft *models.CanvasVersion, err error) (string, bool) {
	if err != nil {
		builder.WriteString("owned_draft: unavailable\n")
		builder.WriteString("snapshot_source: unavailable\n")
		builder.WriteString("nodes: unavailable\n")
		return "", false
	}

	if draft == nil {
		builder.WriteString("owned_draft: none\n")
		return "live", true
	}

	builder.WriteString(fmt.Sprintf("owned_draft_version_id: %s\n", draft.ID.String()))
	builder.WriteString(fmt.Sprintf("owned_draft_display_name: %s\n", draft.DisplayName))
	return "draft", true
}

const noActiveDraftStatus = "[Draft Status]\nNo active drafts. If you recently created a draft and it is no longer here, it was discarded by the user. Your changes were NOT published."

func draftCreatedAt(draft models.CanvasVersion) string {
	if draft.CreatedAt == nil {
		return "unknown"
	}

	return draft.CreatedAt.UTC().Format(time.RFC3339)
}

func wasRecentlyPublished(publishedAt *time.Time) bool {
	return publishedAt != nil && time.Since(*publishedAt) < 10*time.Minute
}

func sessionTitle(organizationID, canvasID uuid.UUID) string {
	org, err := models.FindOrganizationByID(organizationID.String())
	if err != nil {
		return ""
	}
	canvas, err := models.FindCanvas(organizationID, canvasID)
	if err != nil {
		return org.Name
	}
	return org.Name + " - " + canvas.Name
}
