package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

const updateDraftActionName = "update_draft"

type updateDraftAction struct {
	deps Dependencies
}

func newUpdateDraftAction(deps Dependencies) updateDraftAction {
	return updateDraftAction{deps: deps}
}

func (a updateDraftAction) Name() string {
	return updateDraftActionName
}

func (a updateDraftAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	userID := uuid.MustParse(session.UserID)
	draft, err := resolveTargetDraftVersion(canvasID, userID, input)
	if err != nil {
		return updateResult{}, fmt.Errorf("ensure draft: %w", err)
	}

	operations := []*pb.CanvasRepositoryFileOperation{}
	hasCanvasUpdate := strings.TrimSpace(input.CanvasYAML) != ""
	hasConsoleUpdate := strings.TrimSpace(input.ConsoleYAML) != ""
	if hasCanvasUpdate {
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    canvasRepository.CanvasYAMLRepositoryPath,
			Content: []byte(input.CanvasYAML),
		})
	}
	if hasConsoleUpdate {
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    canvasRepository.ConsoleYAMLRepositoryPath,
			Content: []byte(input.ConsoleYAML),
		})
	}
	if len(operations) == 0 {
		return updateResult{}, fmt.Errorf("canvas_yaml or console_yaml is required for update_draft")
	}

	// Validate edits up front with the same parse + registry validation the
	// commit path runs. Staging stores content verbatim, so rejecting malformed
	// YAML here keeps invalid content out of the staging layer the UI reads.
	var nodes []models.Node
	var edges []models.Edge
	if hasCanvasUpdate {
		nodes, edges, err = canvasRepository.ParseAndValidateCanvasYAML(a.deps.Registry, session.OrganizationID, input.CanvasYAML)
		if err != nil {
			return updateResult{}, err
		}
	}
	if hasConsoleUpdate {
		if err := canvasRepository.ValidateConsoleYAML(input.ConsoleYAML); err != nil {
			return updateResult{}, err
		}
	}

	// Write to the UI staging layer instead of committing into the draft version
	// row. This mirrors how the UI editor saves edits: the agent's changes become
	// the user's pending staged edits, which the user then reviews, commits, and
	// publishes. The `read` action reads from the same staging layer, so the
	// agent observes exactly what it staged.
	if _, err := canvasRepository.StageRepositorySpecFileOperations(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		operations,
	); err != nil {
		return updateResult{}, err
	}

	// Auto-layout runs against the staged canvas.yaml and re-stages the
	// positioned result, matching the UI's layout-on-save behavior. The agent
	// never computes node positions itself.
	if layout := resolveCustomToolAutoLayout(input.AutoLayout, hasCanvasUpdate); layout != nil {
		if _, err := canvasRepository.ApplyCanvasAutoLayout(
			ctx,
			session.OrganizationID,
			session.CanvasID,
			draft.ID.String(),
			layout,
		); err != nil {
			return updateResult{}, err
		}
	}

	// Console-only updates leave the staged graph untouched, so summarize the
	// effective staged canvas to keep node/edge counts accurate.
	if !hasCanvasUpdate {
		canvasYAML, readErr := canvasRepository.ReadRepositorySpecFileStaged(
			ctx,
			session.OrganizationID,
			session.CanvasID,
			draft.ID.String(),
			canvasRepository.CanvasYAMLRepositoryPath,
		)
		if readErr != nil {
			return updateResult{}, fmt.Errorf("read staged canvas yaml: %w", readErr)
		}
		nodes, edges, err = canvasRepository.ParseAndValidateCanvasYAML(a.deps.Registry, session.OrganizationID, canvasYAML)
		if err != nil {
			return updateResult{}, fmt.Errorf("summarize staged canvas: %w", err)
		}
	}

	return updateResult{
		Action:     "update_draft",
		CanvasID:   session.CanvasID,
		VersionID:  draft.ID.String(),
		Draft:      draftResult{VersionID: draft.ID.String(), DisplayName: draft.DisplayName, BranchName: draft.GitBranch},
		NodeIssues: collectNodeIssues(nodes),
		Summary:    summarizeParsedCanvas(draft.Name, nodes, edges),
	}, nil
}

func ownedDraftVersions(canvasID, userID uuid.UUID) ([]models.CanvasVersion, error) {
	drafts, err := models.ListDraftCanvasVersions(canvasID)
	if err != nil {
		return nil, err
	}
	owned := make([]models.CanvasVersion, 0, len(drafts))
	for i := range drafts {
		if models.IsUserOwnedDraftVersion(&drafts[i], userID) && models.IsRegisteredDraftVersion(&drafts[i]) {
			owned = append(owned, drafts[i])
		}
	}
	return owned, nil
}

func resolveTargetDraftVersion(canvasID, userID uuid.UUID, input Input) (*models.CanvasVersion, error) {
	requested, err := requestedDraftVersionID(input)
	if err != nil {
		return nil, err
	}
	if requested == uuid.Nil {
		return nil, fmt.Errorf("version_id is required for update_draft; pass the version_id returned by read, create_draft, or the previous update_draft; if read returned live with no version_id, call create_draft first")
	}
	return validatedOwnedDraftVersion(canvasID, userID, requested)
}

func resolveReadableDraftVersion(canvasID, userID uuid.UUID, input Input) (*models.CanvasVersion, error) {
	requested, err := requestedDraftVersionID(input)
	if err != nil {
		return nil, err
	}
	if requested != uuid.Nil {
		return validatedOwnedDraftVersion(canvasID, userID, requested)
	}

	drafts, err := ownedDraftVersions(canvasID, userID)
	if err != nil {
		return nil, err
	}
	switch len(drafts) {
	case 0:
		return nil, nil
	case 1:
		return &drafts[0], nil
	default:
		return nil, fmt.Errorf("multiple owned drafts exist for this app; pass version_id to read a specific draft or use use_draft=false to read live")
	}
}

func validatedOwnedDraftVersion(canvasID, userID, versionID uuid.UUID) (*models.CanvasVersion, error) {
	draft, err := models.FindCanvasVersion(canvasID, versionID)
	if err != nil {
		return nil, fmt.Errorf("load draft version %s: %w", versionID, err)
	}
	if draft.State != models.CanvasVersionStateDraft {
		return nil, fmt.Errorf("draft version %s is not a draft", versionID)
	}
	if !models.IsRegisteredDraftVersion(draft) {
		return nil, fmt.Errorf("draft version %s is not a registered draft branch", versionID)
	}
	if !models.IsUserOwnedDraftVersion(draft, userID) {
		return nil, fmt.Errorf("draft version %s does not belong to the current user", versionID)
	}
	return draft, nil
}

func requestedDraftVersionID(input Input) (uuid.UUID, error) {
	versionID := strings.TrimSpace(input.VersionID)
	draftVersionID := strings.TrimSpace(input.DraftVersionID)
	if versionID != "" && draftVersionID != "" && versionID != draftVersionID {
		return uuid.Nil, fmt.Errorf("version_id and draft_version_id must match when both are provided")
	}

	requested := versionID
	if requested == "" {
		requested = draftVersionID
	}
	if requested == "" {
		return uuid.Nil, nil
	}

	parsed, err := uuid.Parse(requested)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid draft version id: %w", err)
	}
	return parsed, nil
}

func resolveCustomToolAutoLayout(input *AutoLayoutInput, hasCanvasUpdate bool) *pb.CanvasAutoLayout {
	if input == nil || !hasCanvasUpdate {
		return nil
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
