package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcCanvases "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const SuperPlaneCanvasToolName = "superplane_canvas"

type SuperPlaneCanvasTool struct {
	encryptor      crypto.Encryptor
	registry       *registry.Registry
	webhookBaseURL string
	authService    authorization.Authorization
}

type SuperPlaneCanvasToolOptions struct {
	Encryptor      crypto.Encryptor
	Registry       *registry.Registry
	WebhookBaseURL string
	AuthService    authorization.Authorization
}

func NewSuperPlaneCanvasTool(opts SuperPlaneCanvasToolOptions) *SuperPlaneCanvasTool {
	return &SuperPlaneCanvasTool{
		encryptor:      opts.Encryptor,
		registry:       opts.Registry,
		webhookBaseURL: opts.WebhookBaseURL,
		authService:    opts.AuthService,
	}
}

func (t *SuperPlaneCanvasTool) CustomToolName() string {
	return SuperPlaneCanvasToolName
}

func (t *SuperPlaneCanvasTool) ExecuteCustomTool(ctx context.Context, session agents.AgentSessionContext, toolUse agents.CustomToolUse) agents.CustomToolResult {
	if toolUse.Name != SuperPlaneCanvasToolName {
		return customToolError(toolUse.ID, fmt.Sprintf("unsupported custom tool %q", toolUse.Name))
	}

	var input superPlaneCanvasToolInput
	if err := json.Unmarshal([]byte(toolUse.Input), &input); err != nil {
		return customToolError(toolUse.ID, fmt.Sprintf("invalid input: %v", err))
	}
	if err := t.validateSessionBoundInput(session, input.CanvasID); err != nil {
		return customToolError(toolUse.ID, err.Error())
	}

	authedCtx := authentication.SetUserIdInMetadata(ctx, session.UserID)
	var payload any
	var err error

	switch strings.TrimSpace(input.Action) {
	case "read":
		payload, err = t.read(authedCtx, session, input)
	case "update_draft":
		payload, err = t.updateDraft(authedCtx, session, input)
	case "list_integrations":
		payload, err = t.listIntegrations(session)
	default:
		err = fmt.Errorf("unsupported action %q", input.Action)
	}
	if err != nil {
		return customToolError(toolUse.ID, err.Error())
	}

	content, err := json.Marshal(payload)
	if err != nil {
		return customToolError(toolUse.ID, fmt.Sprintf("encode result: %v", err))
	}

	return agents.CustomToolResult{
		CustomToolUseID: toolUse.ID,
		Content:         string(content),
	}
}

func (t *SuperPlaneCanvasTool) validateSessionBoundInput(session agents.AgentSessionContext, requestedCanvasID string) error {
	if strings.TrimSpace(session.CanvasID) == "" || strings.TrimSpace(session.OrganizationID) == "" || strings.TrimSpace(session.UserID) == "" {
		return fmt.Errorf("agent session context is incomplete")
	}
	requestedCanvasID = strings.TrimSpace(requestedCanvasID)
	if requestedCanvasID != "" && requestedCanvasID != session.CanvasID {
		return fmt.Errorf("canvas_id %q is outside this agent session", requestedCanvasID)
	}
	return nil
}

func (t *SuperPlaneCanvasTool) read(ctx context.Context, session agents.AgentSessionContext, input superPlaneCanvasToolInput) (superPlaneCanvasReadResult, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return superPlaneCanvasReadResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(session.OrganizationID), canvasID)
	if err != nil {
		return superPlaneCanvasReadResult{}, fmt.Errorf("load canvas: %w", err)
	}

	draft, err := ownedDraftVersion(canvasID, uuid.MustParse(session.UserID))
	if err != nil {
		return superPlaneCanvasReadResult{}, fmt.Errorf("load draft: %w", err)
	}

	versionID := ""
	source := "live"
	if input.UseDraft == nil || *input.UseDraft {
		if draft != nil {
			versionID = draft.ID.String()
			source = "draft"
		}
	}

	canvasYAML, err := grpcCanvases.ReadRepositorySpecFile(ctx, session.OrganizationID, session.CanvasID, versionID, grpcCanvases.CanvasYAMLRepositoryPath)
	if err != nil {
		return superPlaneCanvasReadResult{}, fmt.Errorf("read canvas yaml: %w", err)
	}

	result := superPlaneCanvasReadResult{
		Action:     "read",
		CanvasID:   session.CanvasID,
		Source:     source,
		VersionID:  versionID,
		Summary:    summarizeCanvasVersion(canvas, selectedVersion(canvas, draft, source)),
		CanvasYAML: canvasYAML,
	}

	if draft != nil {
		result.Draft = &superPlaneCanvasDraftResult{
			VersionID:   draft.ID.String(),
			DisplayName: draft.DisplayName,
			BranchName:  stringValue(draft.BranchName),
		}
	}

	if input.IncludeConsole {
		consoleYAML, consoleErr := grpcCanvases.ReadRepositorySpecFile(ctx, session.OrganizationID, session.CanvasID, versionID, grpcCanvases.ConsoleYAMLRepositoryPath)
		if consoleErr != nil {
			return superPlaneCanvasReadResult{}, fmt.Errorf("read console yaml: %w", consoleErr)
		}
		result.ConsoleYAML = consoleYAML
	}

	if input.IncludeIntegrations {
		integrations, integrationsErr := t.connectedIntegrations(uuid.MustParse(session.OrganizationID))
		if integrationsErr != nil {
			return superPlaneCanvasReadResult{}, integrationsErr
		}
		result.Integrations = integrations
	}

	return result, nil
}

func (t *SuperPlaneCanvasTool) updateDraft(ctx context.Context, session agents.AgentSessionContext, input superPlaneCanvasToolInput) (superPlaneCanvasUpdateResult, error) {
	if t.encryptor == nil || t.registry == nil || t.authService == nil {
		return superPlaneCanvasUpdateResult{}, fmt.Errorf("custom tool executor is missing canvas update dependencies")
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return superPlaneCanvasUpdateResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	draft, err := ensureOwnedDraftVersion(canvasID, uuid.MustParse(session.UserID))
	if err != nil {
		return superPlaneCanvasUpdateResult{}, fmt.Errorf("ensure draft: %w", err)
	}

	operations := []*pb.CanvasRepositoryFileOperation{}
	hasCanvasUpdate := strings.TrimSpace(input.CanvasYAML) != ""
	if hasCanvasUpdate {
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    grpcCanvases.CanvasYAMLRepositoryPath,
			Content: []byte(input.CanvasYAML),
		})
	}
	if strings.TrimSpace(input.ConsoleYAML) != "" {
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    grpcCanvases.ConsoleYAMLRepositoryPath,
			Content: []byte(input.ConsoleYAML),
		})
	}
	if len(operations) == 0 {
		return superPlaneCanvasUpdateResult{}, fmt.Errorf("canvas_yaml or console_yaml is required for update_draft")
	}

	if err := grpcCanvases.ApplyRepositorySpecFileOperations(
		ctx,
		nil,
		t.encryptor,
		t.registry,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		t.webhookBaseURL,
		t.authService,
		resolveCustomToolAutoLayout(input.AutoLayout, hasCanvasUpdate),
		operations,
	); err != nil {
		return superPlaneCanvasUpdateResult{}, err
	}

	updated, err := models.FindCanvasVersion(canvasID, draft.ID)
	if err != nil {
		return superPlaneCanvasUpdateResult{}, fmt.Errorf("load updated draft: %w", err)
	}

	return superPlaneCanvasUpdateResult{
		Action:     "update_draft",
		CanvasID:   session.CanvasID,
		VersionID:  updated.ID.String(),
		Draft:      superPlaneCanvasDraftResult{VersionID: updated.ID.String(), DisplayName: updated.DisplayName, BranchName: stringValue(updated.BranchName)},
		NodeIssues: collectNodeIssues(updated.Nodes),
		Summary:    summarizeCanvasVersion(nil, updated),
	}, nil
}

func (t *SuperPlaneCanvasTool) listIntegrations(session agents.AgentSessionContext) (superPlaneCanvasIntegrationsResult, error) {
	orgID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return superPlaneCanvasIntegrationsResult{}, fmt.Errorf("invalid session organization id: %w", err)
	}
	integrations, err := t.connectedIntegrations(orgID)
	if err != nil {
		return superPlaneCanvasIntegrationsResult{}, err
	}
	return superPlaneCanvasIntegrationsResult{
		Action:       "list_integrations",
		CanvasID:     session.CanvasID,
		Integrations: integrations,
	}, nil
}

func (t *SuperPlaneCanvasTool) connectedIntegrations(orgID uuid.UUID) ([]superPlaneCanvasIntegrationResult, error) {
	integrations, err := models.ListIntegrations(orgID)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}

	result := make([]superPlaneCanvasIntegrationResult, 0, len(integrations))
	for _, integration := range integrations {
		result = append(result, superPlaneCanvasIntegrationResult{
			ID:     integration.ID.String(),
			Name:   integration.InstallationName,
			Vendor: integration.AppName,
			State:  integration.State,
		})
	}
	return result, nil
}

func ownedDraftVersion(canvasID, userID uuid.UUID) (*models.CanvasVersion, error) {
	drafts, err := models.ListDraftCanvasVersions(canvasID)
	if err != nil {
		return nil, err
	}
	for i := range drafts {
		if models.IsUserOwnedDraftVersion(&drafts[i], userID) && models.IsRegisteredDraftVersion(&drafts[i]) {
			return &drafts[i], nil
		}
	}
	return nil, nil
}

func ensureOwnedDraftVersion(canvasID, userID uuid.UUID) (*models.CanvasVersion, error) {
	if draft, err := ownedDraftVersion(canvasID, userID); err != nil || draft != nil {
		return draft, err
	}

	var draft *models.CanvasVersion
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		created, createErr := models.CreateDraftBranchFromLiveInTransaction(tx, canvasID, userID, "", nil, nil)
		draft = created
		return createErr
	})
	if err != nil {
		return nil, err
	}
	return draft, nil
}

func selectedVersion(canvas *models.Canvas, draft *models.CanvasVersion, source string) *models.CanvasVersion {
	if source == "draft" {
		return draft
	}
	if canvas == nil || canvas.LiveVersionID == nil {
		return nil
	}
	version, err := models.FindCanvasVersion(canvas.ID, *canvas.LiveVersionID)
	if err != nil {
		return nil
	}
	return version
}

func summarizeCanvasVersion(canvas *models.Canvas, version *models.CanvasVersion) superPlaneCanvasSummary {
	summary := superPlaneCanvasSummary{}
	if canvas != nil {
		summary.CanvasName = canvas.Name
	}
	if version == nil {
		return summary
	}
	if summary.CanvasName == "" {
		summary.CanvasName = version.Name
	}
	summary.NodeCount = len(version.Nodes)
	summary.EdgeCount = len(version.Edges)
	summary.Nodes = summarizeNodes(version.Nodes, 20)
	return summary
}

func summarizeNodes(nodes []models.Node, limit int) []superPlaneCanvasNodeSummary {
	count := len(nodes)
	if count > limit {
		count = limit
	}
	result := make([]superPlaneCanvasNodeSummary, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, superPlaneCanvasNodeSummary{
			ID:        nodes[i].ID,
			Name:      nodes[i].Name,
			Type:      nodes[i].Type,
			Component: nodeRefName(nodes[i].Ref),
			Issue:     firstNonEmptyString(stringPtrValue(nodes[i].ErrorMessage), stringPtrValue(nodes[i].WarningMessage)),
		})
	}
	return result
}

func collectNodeIssues(nodes []models.Node) []superPlaneCanvasNodeIssue {
	issues := []superPlaneCanvasNodeIssue{}
	for _, node := range nodes {
		if node.ErrorMessage != nil && strings.TrimSpace(*node.ErrorMessage) != "" {
			issues = append(issues, superPlaneCanvasNodeIssue{NodeID: node.ID, NodeName: node.Name, Severity: "error", Message: strings.TrimSpace(*node.ErrorMessage)})
		}
		if node.WarningMessage != nil && strings.TrimSpace(*node.WarningMessage) != "" {
			issues = append(issues, superPlaneCanvasNodeIssue{NodeID: node.ID, NodeName: node.Name, Severity: "warning", Message: strings.TrimSpace(*node.WarningMessage)})
		}
	}
	return issues
}

func resolveCustomToolAutoLayout(input *superPlaneCanvasAutoLayoutInput, hasCanvasUpdate bool) *pb.CanvasAutoLayout {
	if input == nil {
		if !hasCanvasUpdate {
			return nil
		}
		return &pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
			Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
		}
	}

	layout := &pb.CanvasAutoLayout{
		Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
		NodeIds:   append([]string(nil), input.NodeIDs...),
	}

	switch strings.TrimSpace(input.Scope) {
	case "full_canvas", "full-canvas":
		layout.Scope = pb.CanvasAutoLayout_SCOPE_FULL_CANVAS
	case "connected_component", "connected-component":
		layout.Scope = pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT
	}

	return layout
}

func nodeRefName(ref models.NodeRef) string {
	switch {
	case ref.Component != nil:
		return ref.Component.Name
	case ref.Trigger != nil:
		return ref.Trigger.Name
	case ref.Blueprint != nil:
		return ref.Blueprint.ID
	case ref.Widget != nil:
		return ref.Widget.Name
	default:
		return ""
	}
}

func customToolError(toolUseID, message string) agents.CustomToolResult {
	content, _ := json.Marshal(map[string]string{"error": message})
	return agents.CustomToolResult{
		CustomToolUseID: toolUseID,
		Content:         string(content),
		IsError:         true,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type superPlaneCanvasToolInput struct {
	Action              string                           `json:"action"`
	CanvasID            string                           `json:"canvas_id,omitempty"`
	UseDraft            *bool                            `json:"use_draft,omitempty"`
	IncludeConsole      bool                             `json:"include_console,omitempty"`
	IncludeIntegrations bool                             `json:"include_integrations,omitempty"`
	CanvasYAML          string                           `json:"canvas_yaml,omitempty"`
	ConsoleYAML         string                           `json:"console_yaml,omitempty"`
	AutoLayout          *superPlaneCanvasAutoLayoutInput `json:"auto_layout,omitempty"`
}

type superPlaneCanvasAutoLayoutInput struct {
	Scope   string   `json:"scope,omitempty"`
	NodeIDs []string `json:"node_ids,omitempty"`
}

type superPlaneCanvasReadResult struct {
	Action       string                              `json:"action"`
	CanvasID     string                              `json:"canvas_id"`
	Source       string                              `json:"source"`
	VersionID    string                              `json:"version_id,omitempty"`
	Draft        *superPlaneCanvasDraftResult        `json:"draft,omitempty"`
	Summary      superPlaneCanvasSummary             `json:"summary"`
	CanvasYAML   string                              `json:"canvas_yaml"`
	ConsoleYAML  string                              `json:"console_yaml,omitempty"`
	Integrations []superPlaneCanvasIntegrationResult `json:"integrations,omitempty"`
}

type superPlaneCanvasUpdateResult struct {
	Action     string                      `json:"action"`
	CanvasID   string                      `json:"canvas_id"`
	VersionID  string                      `json:"version_id"`
	Draft      superPlaneCanvasDraftResult `json:"draft"`
	Summary    superPlaneCanvasSummary     `json:"summary"`
	NodeIssues []superPlaneCanvasNodeIssue `json:"node_issues,omitempty"`
}

type superPlaneCanvasIntegrationsResult struct {
	Action       string                              `json:"action"`
	CanvasID     string                              `json:"canvas_id"`
	Integrations []superPlaneCanvasIntegrationResult `json:"integrations"`
}

type superPlaneCanvasDraftResult struct {
	VersionID   string `json:"version_id"`
	DisplayName string `json:"display_name,omitempty"`
	BranchName  string `json:"branch_name,omitempty"`
}

type superPlaneCanvasSummary struct {
	CanvasName string                        `json:"canvas_name,omitempty"`
	NodeCount  int                           `json:"node_count"`
	EdgeCount  int                           `json:"edge_count"`
	Nodes      []superPlaneCanvasNodeSummary `json:"nodes,omitempty"`
}

type superPlaneCanvasNodeSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Component string `json:"component,omitempty"`
	Issue     string `json:"issue,omitempty"`
}

type superPlaneCanvasNodeIssue struct {
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type superPlaneCanvasIntegrationResult struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Vendor string `json:"vendor"`
	State  string `json:"state"`
}
