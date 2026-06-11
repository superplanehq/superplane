package actions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
)

const readActionName = "read"

type readAction struct{}

func (readAction) Name() string {
	return readActionName
}

func (readAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return readResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(session.OrganizationID), canvasID)
	if err != nil {
		return readResult{}, fmt.Errorf("load canvas: %w", err)
	}

	draft, err := ownedDraftVersion(canvasID, uuid.MustParse(session.UserID))
	if err != nil {
		return readResult{}, fmt.Errorf("load draft: %w", err)
	}

	versionID := ""
	source := "live"
	if input.UseDraft == nil || *input.UseDraft {
		if draft != nil {
			versionID = draft.ID.String()
			source = "draft"
		}
	}

	canvasYAML, err := canvasRepository.ReadRepositorySpecFile(ctx, session.OrganizationID, session.CanvasID, versionID, canvasRepository.CanvasYAMLRepositoryPath)
	if err != nil {
		return readResult{}, fmt.Errorf("read canvas yaml: %w", err)
	}

	version, err := selectedVersion(canvas, draft, source)
	if err != nil {
		return readResult{}, err
	}

	result := readResult{
		Action:     "read",
		CanvasID:   session.CanvasID,
		Source:     source,
		VersionID:  versionID,
		Summary:    summarizeCanvasVersion(canvas, version),
		CanvasYAML: canvasYAML,
	}

	if draft != nil {
		result.Draft = &draftResult{
			VersionID:   draft.ID.String(),
			DisplayName: draft.DisplayName,
			BranchName:  stringValue(draft.BranchName),
		}
	}

	if input.IncludeConsole {
		consoleYAML, consoleErr := canvasRepository.ReadRepositorySpecFile(ctx, session.OrganizationID, session.CanvasID, versionID, canvasRepository.ConsoleYAMLRepositoryPath)
		if consoleErr != nil {
			return readResult{}, fmt.Errorf("read console yaml: %w", consoleErr)
		}
		result.ConsoleYAML = consoleYAML
	}

	if input.IncludeIntegrations {
		integrations, integrationsErr := listConnectedIntegrations(uuid.MustParse(session.OrganizationID))
		if integrationsErr != nil {
			return readResult{}, integrationsErr
		}
		result.Integrations = integrations
	}

	return result, nil
}

func selectedVersion(canvas *models.Canvas, draft *models.CanvasVersion, source string) (*models.CanvasVersion, error) {
	if source == "draft" {
		return draft, nil
	}
	if canvas == nil || canvas.LiveVersionID == nil {
		return nil, nil
	}
	version, err := models.FindCanvasVersion(canvas.ID, *canvas.LiveVersionID)
	if err != nil {
		return nil, fmt.Errorf("load live canvas version summary: %w", err)
	}
	return version, nil
}
