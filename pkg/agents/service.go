package agents

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
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

const agentTokenTTL = 1 * time.Hour

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

// provisionSession serialises find-or-create across replicas via a Postgres
// advisory lock so two concurrent callers can't both provision an upstream
// session and leak one when the unique index rejects the loser.
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

		upstream, err := s.provider.CreateSession(ctx, CreateSessionOptions{})
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

// ListMessages returns up to `limit` messages older than `beforeID` (or the
// most recent `limit` when `beforeID` is uuid.Nil), chronologically ordered.
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

func (s *Service) SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string) (*models.AgentSessionMessage, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	session, err := models.FindAgentSessionForUser(organizationID, userID, sessionID)
	if err != nil {
		return nil, err
	}

	preamble, err := s.buildPreambleIfFirstTurn(session, organizationID, userID)
	if err != nil {
		return nil, fmt.Errorf("build preamble: %w", err)
	}

	// Send to provider BEFORE persisting the user message. If we persisted
	// first and the provider call failed, the message count would jump from
	// 0 to 1 and the next retry would skip the preamble — leaving the agent
	// without its API token forever (the unique index blocks starting over).
	if err := s.provider.SendMessage(ctx, session.ProviderSessionID, content, SendMessageOptions{ContextPreamble: preamble}); err != nil {
		_ = models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusFailed)
		return nil, fmt.Errorf("forward to provider: %w", err)
	}

	persisted := &models.AgentSessionMessage{
		SessionID: sessionID,
		Role:      models.AgentMessageRoleUser,
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

func (s *Service) buildPreambleIfFirstTurn(session *models.AgentSession, organizationID, userID uuid.UUID) (string, error) {
	count, err := models.CountAgentSessionMessagesInTransaction(database.Conn(), session.ID)
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", nil
	}
	token, _, err := s.mintAgentToken(organizationID.String(), userID.String(), session.CanvasID.String())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"[SuperPlane session context]\ncanvas_id: %s\norganization_id: %s\napi_base_url: %s\napi_token: %s",
		session.CanvasID.String(),
		session.OrganizationID.String(),
		s.baseURL,
		token,
	), nil
}

// enqueueStream wakes a worker to drive the upstream stream for the next turn.
// On failure, the provider has already accepted the message, so we flip the
// session to failed and notify the UI rather than leave it stuck in
// "streaming" until the cleanup loop catches it.
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
			{ResourceType: "canvases", Action: "update", Resources: []string{canvasID}},
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
