package agents

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrSessionForbidden = errors.New("agent session is owned by another user")

type Service struct {
	provider Provider
	auth     authorization.Authorization
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
		refreshed, err := s.refreshStaleProviderSession(ctx, existing)
		if errors.Is(err, ErrSessionBusy) {
			return existing, nil
		}
		return refreshed, err
	}
	return s.provisionSession(ctx, organizationID, userID, canvasID)
}

// ResetSession deletes the current session (and its messages) for the given
// canvas and provisions a brand-new session on the provider.
//
// This is used by the `/clear` UX to give the user a fresh chat with no rewind.
func (s *Service) ResetSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	if err := s.checkAgentPermission(ctx, userID.String(), organizationID.String()); err != nil {
		return nil, err
	}

	upstream, err := s.provider.CreateSession(ctx, CreateSessionOptions{
		Title: sessionTitle(organizationID, canvasID),
	})
	if err != nil {
		return nil, fmt.Errorf("create provider session: %w", err)
	}

	var (
		session            *models.AgentSession
		oldProviderSession string
	)
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", ensureSessionLockKey(organizationID, userID, canvasID)).Error; err != nil {
			return err
		}

		existing, err := findCanvasSession(tx, organizationID, userID, canvasID)
		if err != nil {
			return err
		}
		if existing != nil {
			oldProviderSession = existing.ProviderSessionID
			if err := models.DeleteAgentSessionForUserCanvas(tx, organizationID, userID, canvasID); err != nil {
				return err
			}
		}

		now := time.Now()
		session = &models.AgentSession{
			OrganizationID:          organizationID,
			UserID:                  userID,
			CanvasID:                canvasID,
			Provider:                s.provider.Name(),
			ProviderSessionID:       upstream.ProviderSessionID,
			AgentToolSchemaRevision: s.currentToolSchemaRevision(),
			Status:                  models.AgentSessionStatusIdle,
			// Set so new sessions never trigger the rewind path.
			ContextReplayedAt: &now,
		}
		return models.CreateAgentSessionInTransaction(tx, session)
	})
	if err != nil {
		s.cleanupProviderSession(ctx, upstream.ProviderSessionID)
		return nil, fmt.Errorf("reset session: %w", err)
	}

	// Best-effort: stop / archive the old provider session to avoid wasted work.
	if strings.TrimSpace(oldProviderSession) != "" {
		_ = s.provider.InterruptSession(ctx, oldProviderSession)
		s.archiveProviderSession(ctx, oldProviderSession)
	}

	return session, nil
}

func (s *Service) provisionSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	var session *models.AgentSession
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
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
		upstream, err := s.provider.CreateSession(ctx, CreateSessionOptions{Title: title})
		if err != nil {
			return fmt.Errorf("create provider session: %w", err)
		}

		now := time.Now()
		session = &models.AgentSession{
			OrganizationID:          organizationID,
			UserID:                  userID,
			CanvasID:                canvasID,
			Provider:                s.provider.Name(),
			ProviderSessionID:       upstream.ProviderSessionID,
			AgentToolSchemaRevision: s.currentToolSchemaRevision(),
			Status:                  models.AgentSessionStatusIdle,
			ContextReplayedAt:       &now,
		}
		return models.CreateAgentSessionInTransaction(tx, session)
	})
	if err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
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

	session, err = s.refreshStaleProviderSession(ctx, session)
	if err != nil {
		if errors.Is(err, ErrSessionBusy) {
			return s.handleBusySession(sessionID, organizationID, userID)
		}
		return err
	}

	contextReplayed, err := s.defineOutcomeOnProvider(ctx, session, description, rubric, maxIterations)
	if err != nil {
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
			contextReplayed, err = s.defineOutcomeOnProvider(ctx, recovered, description, rubric, maxIterations)
			if err != nil {
				if errors.Is(err, ErrSessionBusy) {
					return s.handleBusySession(sessionID, organizationID, userID)
				}
				return fmt.Errorf("define outcome after provider session recovery: %w", err)
			}
		} else {
			return fmt.Errorf("define outcome: %w", err)
		}
	}

	if contextReplayed {
		if err := models.MarkAgentSessionContextReplayed(sessionID); err != nil {
			log.WithError(err).WithField("session_id", sessionID).Warn("failed to mark recovered agent context replayed")
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

func (s *Service) defineOutcomeOnProvider(ctx context.Context, session *models.AgentSession, description, rubric string, maxIterations int) (bool, error) {
	description, contextReplayed, err := s.messageWithRewind(session, description)
	if err != nil {
		return false, err
	}

	err = s.provider.DefineOutcome(ctx, session.ProviderSessionID, DefineOutcomeOptions{
		Description:     description,
		Rubric:          rubric,
		MaxIterations:   maxIterations,
		ContextPreamble: s.buildPreamble(session, ModeBuilder),
	})
	return contextReplayed, err
}

func (s *Service) SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string, images []MessageImage, mode ...string) (*models.AgentSessionMessage, error) {
	if content == "" && len(images) == 0 {
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

	session, err = s.refreshStaleProviderSession(ctx, session)
	if err != nil {
		if errors.Is(err, ErrSessionBusy) {
			return nil, s.handleBusySession(sessionID, organizationID, userID)
		}
		return nil, err
	}

	contextReplayed, err := s.sendMessageToProvider(ctx, session, content, images, agentMode)
	if err != nil {
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
			contextReplayed, err = s.sendMessageToProvider(ctx, recovered, content, images, agentMode)
			if err != nil {
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
		Images:    toSessionImages(images),
	}
	if err := models.AppendAgentSessionMessage(persisted); err != nil {
		return nil, fmt.Errorf("persist user message: %w", err)
	}

	if contextReplayed {
		if err := models.MarkAgentSessionContextReplayed(sessionID); err != nil {
			log.WithError(err).WithField("session_id", sessionID).Warn("failed to mark recovered agent context replayed")
		}
	}

	if err := models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusStreaming); err != nil {
		log.WithError(err).Warn("failed to mark agent session as streaming")
	}

	if err := s.enqueueStream(sessionID, organizationID, userID); err != nil {
		return nil, err
	}
	return persisted, nil
}

func toSessionImages(images []MessageImage) datatypes.JSONSlice[models.AgentSessionImage] {
	out := make(datatypes.JSONSlice[models.AgentSessionImage], 0, len(images))
	for _, image := range images {
		out = append(out, models.AgentSessionImage{MediaType: image.MediaType, Data: image.Data})
	}
	return out
}

func (s *Service) sendMessageToProvider(ctx context.Context, session *models.AgentSession, content string, images []MessageImage, mode Mode) (bool, error) {
	message, contextReplayed, err := s.messageWithRewind(session, content)
	if err != nil {
		return false, err
	}

	err = s.provider.SendMessage(ctx, session.ProviderSessionID, message, SendMessageOptions{
		ContextPreamble: s.buildPreamble(session, mode),
		Images:          images,
	})
	return contextReplayed, err
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

func (s *Service) refreshStaleProviderSession(ctx context.Context, session *models.AgentSession) (*models.AgentSession, error) {
	if !s.needsToolSchemaRefresh(session) {
		return session, nil
	}
	if session.Status == models.AgentSessionStatusStreaming {
		return session, nil
	}

	recovered, oldProviderSessionID, replaced, err := s.replaceProviderSession(ctx, session, true)
	if err != nil {
		return nil, fmt.Errorf("refresh stale provider session: %w", err)
	}
	if replaced {
		s.archiveProviderSession(ctx, oldProviderSessionID)
	}
	return recovered, nil
}

func (s *Service) recoverProviderSession(ctx context.Context, stale *models.AgentSession) (*models.AgentSession, error) {
	recovered, _, _, err := s.replaceProviderSession(ctx, stale, false)
	if err != nil {
		return nil, fmt.Errorf("recover provider session: %w", err)
	}
	return recovered, nil
}

type providerSessionRecoveryTarget struct {
	organizationID       uuid.UUID
	canvasID             uuid.UUID
	oldProviderSessionID string
	recovered            *models.AgentSession
}

func (s *Service) replaceProviderSession(ctx context.Context, stale *models.AgentSession, requireStaleToolSchema bool) (*models.AgentSession, string, bool, error) {
	target, err := s.providerSessionRecoveryTarget(stale, requireStaleToolSchema)
	if err != nil {
		return nil, "", false, err
	}
	if target.recovered != nil {
		return target.recovered, "", false, nil
	}

	upstream, err := s.provider.CreateSession(ctx, CreateSessionOptions{
		Title: sessionTitle(target.organizationID, target.canvasID),
	})
	if err != nil {
		return nil, "", false, fmt.Errorf("create provider session: %w", err)
	}

	recovered, err := s.installRecoveredProviderSession(stale, upstream.ProviderSessionID, requireStaleToolSchema)
	if err != nil {
		s.cleanupProviderSession(ctx, upstream.ProviderSessionID)
		return nil, "", false, err
	}
	if recovered.ProviderSessionID != upstream.ProviderSessionID {
		s.cleanupProviderSession(ctx, upstream.ProviderSessionID)
		return recovered, "", false, nil
	}
	return recovered, target.oldProviderSessionID, true, nil
}

func (s *Service) providerSessionRecoveryTarget(stale *models.AgentSession, requireStaleToolSchema bool) (*providerSessionRecoveryTarget, error) {
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
		if requireStaleToolSchema && !s.needsToolSchemaRefresh(locked) {
			target.recovered = locked
			return nil
		}

		target.organizationID = locked.OrganizationID
		target.canvasID = locked.CanvasID
		target.oldProviderSessionID = locked.ProviderSessionID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return target, nil
}

func (s *Service) installRecoveredProviderSession(stale *models.AgentSession, providerSessionID string, requireStaleToolSchema bool) (*models.AgentSession, error) {
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
		if requireStaleToolSchema && !s.needsToolSchemaRefresh(locked) {
			recovered = locked
			return nil
		}

		if err := models.UpdateAgentSessionProviderSessionInTransaction(
			tx,
			locked.ID,
			providerSessionID,
			s.currentToolSchemaRevision(),
			models.AgentSessionStatusIdle,
		); err != nil {
			return err
		}
		locked.ProviderSessionID = providerSessionID
		locked.AgentToolSchemaRevision = s.currentToolSchemaRevision()
		locked.ContextReplayedAt = nil
		locked.Status = models.AgentSessionStatusIdle
		recovered = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	return recovered, nil
}

func (s *Service) needsToolSchemaRefresh(session *models.AgentSession) bool {
	return session.AgentToolSchemaRevision != s.currentToolSchemaRevision()
}

func (s *Service) currentToolSchemaRevision() string {
	revisioner, ok := s.provider.(ProviderToolSchemaRevisioner)
	if !ok {
		return s.provider.Name()
	}
	revision := strings.TrimSpace(revisioner.ToolSchemaRevision())
	if revision == "" {
		return s.provider.Name()
	}
	return revision
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

func (s *Service) archiveProviderSession(ctx context.Context, providerSessionID string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return
	}
	archiver, ok := s.provider.(ProviderSessionArchiver)
	if !ok {
		return
	}
	if err := archiver.ArchiveSession(ctx, providerSessionID); err != nil && !errors.Is(err, ErrProviderSessionUnavailable) {
		log.WithError(err).WithField("provider_session_id", providerSessionID).Warn("failed to archive stale provider session")
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
		result += "These drafts may belong to other sessions or users. To continue a known draft branch, pass its version_id to 'superplane_app' actions 'read' and 'patch_draft'. patch_draft always requires version_id. Use 'create_draft' when read returned live/no version_id, or when the user explicitly wants another draft branch. Do not assume an unrelated draft is yours.\n"
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

const noActiveDraftStatus = "[Draft Status]\nNo active drafts. If you need to edit the app, call 'superplane_app' action 'create_draft' first, then pass the returned version_id to 'patch_draft'. If you recently created a draft and it is no longer here, it was discarded by the user. Your changes were NOT published."

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
