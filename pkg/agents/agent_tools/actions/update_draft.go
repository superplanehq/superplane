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
	if a.deps.Encryptor == nil || a.deps.Registry == nil || a.deps.AuthService == nil {
		return updateResult{}, fmt.Errorf("custom tool executor is missing canvas update dependencies")
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	draft, err := ensureOwnedDraftVersion(canvasID, uuid.MustParse(session.UserID))
	if err != nil {
		return updateResult{}, fmt.Errorf("ensure draft: %w", err)
	}

	operations := []*pb.CanvasRepositoryFileOperation{}
	hasCanvasUpdate := strings.TrimSpace(input.CanvasYAML) != ""
	if hasCanvasUpdate {
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    canvasRepository.CanvasYAMLRepositoryPath,
			Content: []byte(input.CanvasYAML),
		})
	}
	if strings.TrimSpace(input.ConsoleYAML) != "" {
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    canvasRepository.ConsoleYAMLRepositoryPath,
			Content: []byte(input.ConsoleYAML),
		})
	}
	if len(operations) == 0 {
		return updateResult{}, fmt.Errorf("canvas_yaml or console_yaml is required for update_draft")
	}

	if err := canvasRepository.ApplyRepositorySpecFileOperations(
		ctx,
		a.deps.UsageService,
		a.deps.Encryptor,
		a.deps.Registry,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		a.deps.WebhookBaseURL,
		a.deps.AuthService,
		resolveCustomToolAutoLayout(input.AutoLayout, hasCanvasUpdate),
		operations,
	); err != nil {
		return updateResult{}, err
	}

	updated, err := models.FindCanvasVersion(canvasID, draft.ID)
	if err != nil {
		return updateResult{}, fmt.Errorf("load updated draft: %w", err)
	}

	return updateResult{
		Action:     "update_draft",
		CanvasID:   session.CanvasID,
		VersionID:  updated.ID.String(),
		Draft:      draftResult{VersionID: updated.ID.String(), DisplayName: updated.DisplayName, BranchName: stringValue(updated.BranchName)},
		NodeIssues: collectNodeIssues(updated.Nodes),
		Summary:    summarizeCanvasVersion(nil, updated),
	}, nil
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

	return models.CreateDraftBranchFromLive(canvasID, userID, "", nil, nil)
}

func resolveCustomToolAutoLayout(input *AutoLayoutInput, hasCanvasUpdate bool) *pb.CanvasAutoLayout {
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
