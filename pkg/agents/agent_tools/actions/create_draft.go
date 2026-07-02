package actions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/models"
)

const createDraftActionName = "create_draft"

type createDraftAction struct{}

func (createDraftAction) Name() string {
	return createDraftActionName
}

func (createDraftAction) Execute(_ context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(session.OrganizationID), canvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("load canvas: %w", err)
	}
	if canvas.LiveVersionID == nil {
		return updateResult{}, fmt.Errorf("canvas has no live version")
	}

	liveVersion, err := models.FindCanvasVersion(canvasID, *canvas.LiveVersionID)
	if err != nil {
		return updateResult{}, fmt.Errorf("load live version: %w", err)
	}

	return updateResult{
		Action:    createDraftActionName,
		CanvasID:  session.CanvasID,
		VersionID: liveVersion.ID.String(),
		Draft:     draftResult{VersionID: liveVersion.ID.String()},
		Summary:   summarizeCanvasVersion(nil, liveVersion),
	}, nil
}
