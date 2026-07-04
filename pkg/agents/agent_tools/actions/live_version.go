package actions

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

func resolveLiveCanvasVersion(canvasID uuid.UUID, input Input) (*models.CanvasVersion, error) {
	requested, err := requestedVersionID(input)
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

func requestedVersionID(input Input) (uuid.UUID, error) {
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
		return uuid.Nil, fmt.Errorf("invalid version id: %w", err)
	}
	return parsed, nil
}
