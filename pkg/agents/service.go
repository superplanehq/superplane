package agents

import (
	"context"
	"errors"
	"fmt"
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

const chatTitleMaxLength = 60

func deriveChatTitle(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		trimmed = line
		break
	}
	runes := []rune(trimmed)
	if len(runes) > chatTitleMaxLength {
		return string(runes[:chatTitleMaxLength-1]) + "…"
	}
	return trimmed
}

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

func (s *Service) CreateSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error) {
	if err := s.checkAgentPermission(userID.String(), organizationID.String()); err != nil {
		return nil, err
	}

	upstream, err := s.provider.CreateSession(ctx, CreateSessionOptions{})
	if err != nil {
		return nil, fmt.Errorf("create provider session: %w", err)
	}

	session := &models.AgentSession{
		OrganizationID:    organizationID,
		UserID:            userID,
		CanvasID:          canvasID,
		Provider:          s.provider.Name(),
		ProviderSessionID: upstream.ProviderSessionID,
		Status:            models.AgentSessionStatusIdle,
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.CreateAgentSessionInTransaction(tx, session)
	})
	if err != nil {
		return nil, fmt.Errorf("persist session: %w", err)
	}

	return session, nil
}

func (s *Service) ListSessions(organizationID, userID, canvasID uuid.UUID) ([]models.AgentSession, error) {
	return models.ListAgentSessionsForUser(organizationID, userID, canvasID)
}

func (s *Service) GetSession(organizationID, userID, sessionID uuid.UUID) (*models.AgentSession, error) {
	return models.FindAgentSessionForUser(organizationID, userID, sessionID)
}

func (s *Service) ListMessages(sessionID uuid.UUID) ([]models.AgentSessionMessage, error) {
	return models.ListAgentSessionMessages(sessionID)
}

func (s *Service) SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string) (*models.AgentSessionMessage, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	session, err := models.FindAgentSessionForUser(organizationID, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if session.ArchivedAt != nil {
		return nil, ErrSessionAlreadyTerminated
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
			// Mint and inject without persisting — the token never
			// touches the DB.
			token, _, mintErr := s.mintAgentToken(organizationID.String(), userID.String(), session.CanvasID.String())
			if mintErr != nil {
				return mintErr
			}
			preamble = s.firstTurnPreamble(session, token)
			if title := deriveChatTitle(content); title != "" {
				if err := models.UpdateAgentSessionTitleInTransaction(tx, sessionID, title); err != nil {
					return err
				}
			}
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
		log.WithError(err).Error("failed to enqueue agent stream request")
		return nil, fmt.Errorf("enqueue stream: %w", err)
	}

	return persisted, nil
}

// ArchiveSession soft-archives locally even if the upstream archive fails so
// the user is never stuck in a chat we cannot get them out of.
func (s *Service) ArchiveSession(ctx context.Context, organizationID, userID, sessionID uuid.UUID) error {
	session, err := models.FindAgentSessionForUser(organizationID, userID, sessionID)
	if err != nil {
		return err
	}
	if session.ArchivedAt != nil {
		return nil
	}

	if err := s.provider.ArchiveSession(ctx, session.ProviderSessionID); err != nil {
		log.WithError(err).WithField("session_id", sessionID).Warn("failed to archive upstream agent session")
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.ArchiveAgentSessionInTransaction(tx, sessionID)
	})
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
