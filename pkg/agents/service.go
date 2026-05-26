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
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var ErrSessionForbidden = errors.New("agent session is owned by another user")

type Service struct {
	provider  Provider
	auth      authorization.Authorization
	jwtSigner *jwt.Signer
	baseURL   string
	clock     func() time.Time
}

func NewService(provider Provider, auth authorization.Authorization, jwtSigner *jwt.Signer, baseURL string) *Service {
	return &Service{
		provider:  provider,
		auth:      auth,
		jwtSigner: jwtSigner,
		baseURL:   baseURL,
		clock:     time.Now,
	}
}

func (s *Service) ProviderName() string { return s.provider.Name() }

// EnsureSession returns the user's single chat session for the given canvas,
// provisioning it on the upstream provider on first call.
func (s *Service) EnsureSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	if err := s.checkAgentPermission(userID.String(), organizationID.String()); err != nil {
		return nil, err
	}

	existing, err := findCanvasSession(database.Conn(), organizationID, userID, canvasID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return s.provisionSession(ctx, organizationID, userID, canvasID)
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

func (s *Service) InterruptSession(ctx context.Context, organizationID, userID, sessionID uuid.UUID) error {
	session, err := s.GetSession(organizationID, userID, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if err := s.provider.InterruptSession(ctx, session.ProviderSessionID); err != nil {
		return fmt.Errorf("interrupt: %w", err)
	}
	return nil
}

func (s *Service) DefineOutcome(ctx context.Context, organizationID, userID, sessionID uuid.UUID, description, rubric string, maxIterations int) error {
	session, err := s.GetSession(organizationID, userID, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	preamble, err := s.buildPreamble(session, organizationID, userID, ModeBuilder)
	if err != nil {
		return fmt.Errorf("build preamble: %w", err)
	}

	if err := s.provider.DefineOutcome(ctx, session.ProviderSessionID, DefineOutcomeOptions{
		Description:     description,
		Rubric:          rubric,
		MaxIterations:   maxIterations,
		ContextPreamble: preamble,
	}); err != nil {
		return fmt.Errorf("define outcome: %w", err)
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

	preamble, err := s.buildPreamble(session, organizationID, userID, agentMode)
	if err != nil {
		return nil, fmt.Errorf("build preamble: %w", err)
	}

	if err := s.provider.SendMessage(ctx, session.ProviderSessionID, content, SendMessageOptions{ContextPreamble: preamble}); err != nil {
		_ = models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusFailed)
		return nil, fmt.Errorf("forward to provider: %w", err)
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

func (s *Service) buildPreamble(session *models.AgentSession, organizationID, userID uuid.UUID, mode Mode) (string, error) {
	token, expiresAt, err := s.mintAgentToken(organizationID.String(), userID.String(), session.CanvasID.String())
	if err != nil {
		return "", err
	}
	base := fmt.Sprintf(
		preambleTemplate,
		session.CanvasID.String(),
		session.OrganizationID.String(),
		s.baseURL,
		token,
		expiresAt.UTC().Format(time.RFC3339),
		session.CanvasID.String(),
		session.CanvasID.String(),
	)
	draftStatus := getDraftStatus(session.CanvasID)
	return base + "\n\n" + modeInstructions(mode) + "\n\n" + draftStatus, nil
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

// mintAgentToken returns a JWT scoped to one canvas to bound the blast-
// radius if it leaks out of the agent container.
func (s *Service) mintAgentToken(organizationID, userID, canvasID string) (string, time.Time, error) {
	claims := jwt.ScopedTokenClaims{
		Subject: userID,
		OrgID:   organizationID,
		Purpose: "agent-builder",
		Scopes: jwt.ScopesFromPermissions([]jwt.Permission{
			{ResourceType: "org", Action: "read"},
			{ResourceType: "integrations", Action: "read"},
			{ResourceType: "canvases", Action: "read", Resources: []string{canvasID}},
			{ResourceType: "canvases", Action: "update_version", Resources: []string{canvasID}},
		}),
	}
	token, err := s.jwtSigner.GenerateScopedToken(claims, agentTokenTTL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("mint agent token: %w", err)
	}
	return token, s.clock().Add(agentTokenTTL), nil
}

// checkAgentPermission enforces the org-level baseline. Per-canvas access is
// gated by the gRPC handler's models.FindCanvas(orgID, canvasID).
func (s *Service) checkAgentPermission(userID, organizationID string) error {
	checks := []struct{ resource, action string }{
		{"agents", "create"},
		{"canvases", "read"},
	}
	for _, c := range checks {
		allowed, err := s.auth.CheckOrganizationPermission(userID, organizationID, c.resource, c.action)
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
				"- Active draft: version %s (created %s)\n",
				draft.ID.String(),
				draftCreatedAt(draft),
			)
		}
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
