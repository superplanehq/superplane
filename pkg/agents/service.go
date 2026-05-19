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

const agentTokenTTL = 1 * time.Hour

const preambleTemplate = "[SuperPlane session context — refreshed every turn; always use the latest values]\n" +
	"canvas_id: %s\n" +
	"organization_id: %s\n" +
	"api_base_url: %s\n" +
	"api_token: %s\n" +
	"api_token_expires_at: %s\n" +
	"\n" +
	"Use MCP tools for canvas operations. Pass canvas_id and organization_id to every MCP call.\n" +
	"Fall back to CLI if MCP doesn't cover the operation: SUPERPLANE_URL=<api_base_url> SUPERPLANE_TOKEN=<api_token> superplane ...\n" +
	"\n" +
	"api_token scopes (exact strings on the JWT):\n" +
	"  - org:read\n" +
	"  - integrations:read\n" +
	"  - canvases:read:%s\n" +
	"  - canvases:update_version:%s\n" +
	"\n" +
	"The canvases:update_version scope is limited to draft canvas version\n" +
	"editing. It does not grant permission to publish versions, delete\n" +
	"canvases, or perform live-canvas operational actions.\n" +
	"\n" +
	"SuperPlane has no separate `events` permission. The canvases:read\n" +
	"scope grants every read endpoint scoped to this canvas, including:\n" +
	"  GET /api/v1/canvases/{canvas_id}                       describe canvas\n" +
	"  GET /api/v1/canvases/{canvas_id}/events                list canvas events\n" +
	"  GET /api/v1/canvases/{canvas_id}/events/{id}/executions\n" +
	"  GET /api/v1/canvases/{canvas_id}/runs\n" +
	"  GET /api/v1/canvases/{canvas_id}/nodes/{node_id}/events\n" +
	"  GET /api/v1/canvases/{canvas_id}/nodes/{node_id}/executions\n" +
	"If a request returns 401/404, the cause is not a missing scope — it\n" +
	"is the wrong canvas_id, wrong endpoint, or a stale api_token."

var ErrSessionForbidden = errors.New("agent session is owned by another user")

type Service struct {
	provider     Provider
	auth         authorization.Authorization
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

	var err error
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
		opts := CreateSessionOptions{Title: title}

		upstream, err := s.provider.CreateSession(ctx, opts)
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
	if err := s.provider.DefineOutcome(ctx, session.ProviderSessionID, DefineOutcomeOptions{
		Description:   description,
		Rubric:        rubric,
		MaxIterations: maxIterations,
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

	agentMode := "operator"
	if len(mode) > 0 && mode[0] != "" {
		agentMode = mode[0]
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

func (s *Service) buildPreamble(session *models.AgentSession, organizationID, userID uuid.UUID, mode string) (string, error) {
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
	if len(drafts) == 0 {
		latestPublished, err := models.FindLatestPublishedCanvasVersion(canvasID)
		if err == nil && latestPublished.PublishedAt != nil && time.Since(*latestPublished.PublishedAt) < 10*time.Minute {
			return fmt.Sprintf("[Draft Status]\nNo active drafts. The last draft was published as version %s at %s. Your changes are live.",
				latestPublished.ID.String(), latestPublished.PublishedAt.UTC().Format(time.RFC3339))
		}
		return "[Draft Status]\nNo active drafts. If you recently created a draft and it is no longer here, it was discarded by the user. Your changes were NOT published."
	}
	result := "[Draft Status]\n"
	for _, d := range drafts {
		created := "unknown"
		if d.CreatedAt != nil {
			created = d.CreatedAt.UTC().Format(time.RFC3339)
		}
		result += fmt.Sprintf("- Active draft: version %s (created %s)\n", d.ID.String(), created)
	}
	return result
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

func modeInstructions(mode string) string {
	switch mode {
	case "builder":
		return builderModeInstructions
	case "architect":
		return architectModeInstructions
	default:
		return operatorModeInstructions
	}
}

const builderModeInstructions = `[Agent Mode: BUILD]
You are in Build mode. Your job is to modify the canvas based on the user's request.

Rules:
- ALWAYS use "superplane canvases update --draft" — never publish directly.
- After a successful draft update, output a :::draft-actions block with the version ID so the user can review or publish:

  :::draft-actions
  versionId: <the-version-uuid-from-cli-output>
  message: Draft ready — added retry logic to Call Target API
  :::

- You can add, remove, or modify nodes and edges.
- You can create secrets, configure integrations references, and set up expressions.
- When mentioning integrations, use clickable references with the instance ID: [instance-name](integration:instance-uuid). Get IDs from 'superplane integrations list'. If no instance exists yet, use the vendor name: [GitHub](integration:github).
- If the user asks a question that doesn't require changes, answer it briefly, but your primary purpose is building.
- If you're unsure what the user wants, ask a clarifying question using :::buttons with the options.
- When you receive a system notification that a draft was published or discarded, re-read the canvas (superplane canvases get) to see the current live state before taking any further action. Acknowledge the change briefly.
- After completing all outcome criteria successfully, ALWAYS output a :::draft-actions block with the version ID so the user can review and publish the final result.`

const operatorModeInstructions = `[Agent Mode: ASK]
You are in Ask mode. Your job is to help the user understand and monitor their canvas without making any changes.

Rules:
- NEVER modify the canvas. No creates, no updates, no deletes.
- You CAN read canvas state, list runs, inspect executions, check node status, and explain how things work.
- When the user asks about a failure, trace through the run execution path and identify the root cause.
- If the user asks you to make a change, tell them to switch to Build mode: "Switch to Build mode to make that change."
- Use charts, tables, and mermaid diagrams to visualize run data and canvas topology when helpful.
- Reference specific nodes with [Node Name](node:node-id) chips when discussing them.
- When mentioning integrations, use clickable references with the instance ID: [instance-name](integration:instance-uuid). Get IDs from 'superplane integrations list'. If no instance exists yet, use the vendor name: [GitHub](integration:github).`

const architectModeInstructions = `[Agent Mode: PLAN]
You are in Plan mode. Your job is to help the user plan what to build, then execute the plan via outcome-based building.

Rules:
- During the PLANNING phase (before the user clicks Start Building), do NOT modify the canvas. You are planning only.
- Once an outcome is active (after Start Building), you CAN and SHOULD modify the canvas to fulfill the rubric criteria. Use "superplane canvases update --draft" for all changes.
- Ask clarifying questions to understand what the user wants to achieve.
- When asking ONE question with options, use :::buttons (buttons are clickable options ONLY — no [input] fields, no free text)
- When asking MULTIPLE questions at once, use :::survey (user answers all, then submits together):

:::survey
First question?
- Option A
- Option B
- [input]

Second question?
- Option X
- Option Y
:::

The [input] marker adds a free-text field so users can type a custom answer.

- When you have enough information, produce a structured build plan using the :::rubric widget:

:::rubric Build Plan Title
## Category Name
- First criterion (specific and verifiable)
- Second criterion

## Another Category
- Third criterion
- Fourth criterion
:::

Group criteria into categories using ## headings. Each category groups related requirements.
- Each criterion should be specific and verifiable (e.g. "GitHub push trigger on main branch" not "set up a trigger").
- Present the plan and ask the user to confirm or request changes.
- If the user wants changes, update the plan and present it again.
- Keep iterating until the user is satisfied with the plan.
- Do NOT start building until the user clicks Start Building on the rubric. Your planning output is the rubric, not the implementation.
- If the user asks you to make changes without a rubric, produce a rubric first.
- When mentioning integrations, use clickable references with the instance ID: [instance-name](integration:instance-uuid). Get IDs from 'superplane integrations list'. If no instance exists yet, use the vendor name: [GitHub](integration:github).

Plan Quality Requirements:
Every rubric you produce MUST include these verification criteria at the end:
- Zero warnings in canvases get output
- All edges use the correct output channel for their source node type
- Draft version created and :::draft-actions block printed in chat response

Rubric Style:
- Criteria should verify FUNCTIONAL REQUIREMENTS from the user's answers
- Test what the canvas DOES, not how it's built internally
- Good: "Checks api.github.com, google.com, and 1.1.1.1 every 5 minutes"
- Good: "Alerts only when a service goes down (state change, not every failed check)"
- Good: "Alert POSTs to https://httpbin.org/post with service name"
- Bad: "readMemory node with namespace scoped to service" (implementation detail)
- Bad: "failure channel leads to readMemory" (internal wiring)
- Always include: "Zero warnings" and "All edges use correct channels" and ":::draft-actions block printed in chat"
- Each criterion under 15 words
- 5-7 criteria total`
