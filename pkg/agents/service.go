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

	if existing, err := models.FindAgentSessionByCanvasInTransaction(database.Conn(), organizationID, userID, canvasID); err == nil {
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load session: %w", err)
	}

	var session *models.AgentSession
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", ensureSessionLockKey(organizationID, userID, canvasID)).Error; err != nil {
			return err
		}

		if found, err := models.FindAgentSessionByCanvasInTransaction(tx, organizationID, userID, canvasID); err == nil {
			session = found
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
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

func ensureSessionLockKey(organizationID, userID, canvasID uuid.UUID) int64 {
	h := fnv.New64a()
	h.Write(organizationID[:])
	h.Write(userID[:])
	h.Write(canvasID[:])
	return int64(binary.BigEndian.Uint64(h.Sum(nil))) //nolint:gosec // wraparound is fine; we just need a deterministic key
}

func (s *Service) GetSession(organizationID, userID, sessionID uuid.UUID) (*models.AgentSession, error) {
	return models.FindAgentSessionForUser(organizationID, userID, sessionID)
}

// ListMessages returns up to `limit` messages older than `beforeID` (or the
// most recent `limit` when `beforeID` is uuid.Nil), chronologically ordered.
func (s *Service) ListMessages(sessionID, beforeID uuid.UUID, limit int) ([]models.AgentSessionMessage, error) {
	var before *models.AgentSessionMessage
	if beforeID != uuid.Nil {
		var anchor models.AgentSessionMessage
		if err := database.Conn().
			Where("session_id = ?", sessionID).
			Where("id = ?", beforeID).
			First(&anchor).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		before = &anchor
	}
	return models.ListAgentSessionMessagesPage(sessionID, before, limit)
}

func (s *Service) SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string) (*models.AgentSessionMessage, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	session, err := models.FindAgentSessionForUser(organizationID, userID, sessionID)
	if err != nil {
		return nil, err
	}

	preamble := ""
	persisted := &models.AgentSessionMessage{
		SessionID: sessionID,
		Role:      models.AgentMessageRoleUser,
		Content:   content,
	}
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		count, err := models.CountAgentSessionMessagesInTransaction(tx, sessionID)
		if err != nil {
			return err
		}
		if count == 0 {
			token, _, mintErr := s.mintAgentToken(organizationID.String(), userID.String(), session.CanvasID.String())
			if mintErr != nil {
				return mintErr
			}
			preamble = s.firstTurnPreamble(session, token)
		}
		return models.AppendAgentSessionMessageInTransaction(tx, persisted)
	})
	if err != nil {
		return nil, fmt.Errorf("persist user message: %w", err)
	}

	if err := s.provider.SendMessage(ctx, session.ProviderSessionID, content, SendMessageOptions{ContextPreamble: preamble}); err != nil {
		_ = models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusFailed)
		return nil, fmt.Errorf("forward to provider: %w", err)
	}

	if err := models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusStreaming); err != nil {
		log.WithError(err).Warn("failed to mark agent session as streaming")
	}

	if err := messages.PublishAgentStreamRequested(messages.AgentStreamRequest{
		SessionID:      sessionID.String(),
		OrganizationID: organizationID.String(),
		UserID:         userID.String(),
	}); err != nil {
		// The provider has already accepted the message, so its response
		// will go to a worker that no longer exists. Flip the session to
		// failed and emit the matching event so the UI doesn't show a
		// hung "streaming" state until the cleanup loop catches it.
		_ = models.UpdateAgentSessionStatus(sessionID, models.AgentSessionStatusFailed)
		_ = messages.PublishAgentSessionEvent(messages.AgentSessionEventMessage{
			SessionID: sessionID.String(),
			Event:     "session_failed",
			Status:    models.AgentSessionStatusFailed,
			Error:     "failed to enqueue stream",
		})
		log.WithError(err).Error("failed to enqueue agent stream request")
		return nil, fmt.Errorf("enqueue stream: %w", err)
	}
	return persisted, nil
}

func (s *Service) firstTurnPreamble(session *models.AgentSession, token string) string {
	return fmt.Sprintf(
		"[SuperPlane session context]\ncanvas_id: %s\norganization_id: %s\napi_base_url: %s\napi_token: %s",
		session.CanvasID.String(),
		session.OrganizationID.String(),
		s.baseURL,
		token,
	)
}

// mintAgentToken returns a JWT scoped to one canvas to bound the blast-
// radius if it leaks out of the agent container.
func (s *Service) mintAgentToken(organizationID, userID, canvasID string) (string, time.Time, error) {
	permissions := []jwt.Permission{
		{ResourceType: "org", Action: "read"},
		{ResourceType: "integrations", Action: "read"},
		{ResourceType: "canvases", Action: "read", Resources: []string{canvasID}},
		{ResourceType: "canvases", Action: "update", Resources: []string{canvasID}},
	}

	claims := jwt.ScopedTokenClaims{
		Subject: userID,
		OrgID:   organizationID,
		Purpose: "agent-builder",
		Scopes:  jwt.ScopesFromPermissions(permissions),
	}

	token, err := s.jwtSigner.GenerateScopedToken(claims, agentTokenTTL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("mint agent token: %w", err)
	}
	return token, s.clock().Add(agentTokenTTL), nil
}

// checkAgentPermission enforces the org-level baseline. Canvas-level access
// is implicitly enforced by the caller's models.FindCanvas(orgID, canvasID)
// — this codebase's auth model has no per-canvas RBAC layer; org members
// see all of the org's canvases.
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
