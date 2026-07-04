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

type readAction struct {
	deps Dependencies
}

func newReadAction(deps Dependencies) readAction {
	return readAction{deps: deps}
}

func (readAction) Name() string {
	return readActionName
}

func (a readAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return readResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(session.OrganizationID), canvasID)
	if err != nil {
		return readResult{}, fmt.Errorf("load canvas: %w", err)
	}

	liveVersion, err := resolveLiveCanvasVersion(canvasID, input)
	if err != nil {
		return readResult{}, fmt.Errorf("load live version: %w", err)
	}

	versionID := liveVersion.ID.String()

	// Read effective staged content (staged edits when present, live content
	// otherwise) so the agent observes the same state the UI editor uses.
	canvasYAML, err := canvasRepository.ReadRepositorySpecFileStaged(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		versionID,
		canvasRepository.CanvasYAMLRepositoryPath,
	)
	if err != nil {
		return readResult{}, fmt.Errorf("read canvas yaml: %w", err)
	}

	result := readResult{
		Action:            readActionName,
		CanvasID:          session.CanvasID,
		Source:            "staging",
		VersionID:         versionID,
		Summary:           a.summarize(session.OrganizationID, canvas, liveVersion, canvasYAML),
		CanvasYAMLBytes:   len(canvasYAML),
		CanvasYAMLOmitted: !input.IncludeCanvasYAML,
	}

	if input.IncludeCanvasYAML {
		result.CanvasYAML = canvasYAML
	}

	if input.IncludeConsole {
		consoleYAML, consoleErr := canvasRepository.ReadRepositorySpecFileStaged(
			ctx,
			session.OrganizationID,
			session.CanvasID,
			versionID,
			canvasRepository.ConsoleYAMLRepositoryPath,
		)
		if consoleErr != nil {
			return readResult{}, fmt.Errorf("read console yaml: %w", consoleErr)
		}
		result.ConsoleYAML = consoleYAML
	}

	if input.IncludeIntegrations {
		integrations, integrationsErr := listConnectedIntegrations(ctx, uuid.MustParse(session.OrganizationID))
		if integrationsErr != nil {
			return readResult{}, integrationsErr
		}
		result.Integrations = integrations
	}

	return result, nil
}

func (a readAction) summarize(organizationID string, canvas *models.Canvas, version *models.CanvasVersion, canvasYAML string) summary {
	nodes, edges, err := canvasRepository.ParseAndValidateCanvasYAML(a.deps.Registry, organizationID, canvasYAML)
	if err != nil {
		return summarizeCanvasVersion(canvas, version)
	}

	name := ""
	if canvas != nil {
		name = canvas.Name
	}
	return summarizeParsedCanvas(name, nodes, edges)
}
