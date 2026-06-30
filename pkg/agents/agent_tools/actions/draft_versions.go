package actions

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

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
		return nil, draftVersionRequiredError(input.Action)
	}
	return validatedOwnedDraftVersion(canvasID, userID, requested)
}

func draftVersionRequiredError(action string) error {
	switch strings.TrimSpace(action) {
	case patchDraftActionName:
		return fmt.Errorf("version_id is required for patch_draft; pass the version_id returned by read, create_draft, or the previous patch_draft; if read returned live with no version_id, call create_draft first")
	case "":
		return fmt.Errorf("version_id is required; pass the version_id returned by read, create_draft, or the previous draft update; if read returned live with no version_id, call create_draft first")
	default:
		return fmt.Errorf("version_id is required for %s; pass the version_id returned by read, create_draft, or the previous draft update; if read returned live with no version_id, call create_draft first", strings.TrimSpace(action))
	}
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
	if draft.GitBranch == models.CanvasGitBranchMain {
		return nil, fmt.Errorf("version %s is on main, not a branch", versionID)
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

func resolveToolAutoLayoutInput(input *AutoLayoutInput) *pb.CanvasAutoLayout {
	if input == nil {
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
	default:
		if len(layout.NodeIds) > 0 {
			layout.Scope = pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT
		}
	}

	return layout
}
