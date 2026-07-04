package actions

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func ownedDraftVersions(canvasID, userID uuid.UUID) ([]models.CanvasVersion, error) {
	_ = canvasID
	_ = userID
	return nil, nil
}

func resolveTargetDraftVersion(canvasID, userID uuid.UUID, input Input) (*models.CanvasVersion, error) {
	return liveCanvasVersion(canvasID, userID, input)
}

func draftVersionRequiredError(action string) error {
	switch strings.TrimSpace(action) {
	case patchStagingActionName:
		return fmt.Errorf("version_id is optional for patch_staging; edits are staged against the live canvas")
	case "":
		return fmt.Errorf("version_id is optional; edits are staged against the live canvas")
	default:
		return fmt.Errorf("version_id is optional for %s; edits are staged against the live canvas", strings.TrimSpace(action))
	}
}

func liveCanvasVersion(canvasID, userID uuid.UUID, input Input) (*models.CanvasVersion, error) {
	_ = userID
	requested, err := requestedDraftVersionID(input)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	if err != nil {
		return nil, fmt.Errorf("load canvas: %w", err)
	}
	if canvas.LiveVersionID == nil {
		return nil, fmt.Errorf("canvas has no live version")
	}

	liveVersion, err := models.FindCanvasVersion(canvasID, *canvas.LiveVersionID)
	if err != nil {
		return nil, fmt.Errorf("load live version: %w", err)
	}

	if requested != uuid.Nil && requested != liveVersion.ID {
		return nil, fmt.Errorf("version_id %s is not the current live version", requested)
	}

	return liveVersion, nil
}

func validatedOwnedDraftVersion(canvasID, userID, versionID uuid.UUID) (*models.CanvasVersion, error) {
	input := Input{VersionID: versionID.String()}
	return liveCanvasVersion(canvasID, userID, input)
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
