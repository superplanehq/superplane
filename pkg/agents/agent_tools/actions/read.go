package actions

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/services/files"
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

	db := database.DB(ctx)
	canvas, err := models.FindCanvasInTransaction(db, uuid.MustParse(session.OrganizationID), canvasID)
	if err != nil {
		return readResult{}, fmt.Errorf("load canvas: %w", err)
	}

	liveVersion, err := models.FindCanvasVersionInTransaction(db, canvasID, *canvas.LiveVersionID)
	if err != nil {
		return readResult{}, fmt.Errorf("load live version: %w", err)
	}

	// Read effective staged content (staged edits when present, live content
	// otherwise) so the agent observes the same state the UI editor uses.
	fileReader := files.NewAppFileReader(db, canvas, uuid.MustParse(session.UserID))
	reader, err := fileReader.Read(ctx, files.CanvasYAMLPath)
	if err != nil {
		return readResult{}, fmt.Errorf("read canvas yaml: %w", err)
	}
	canvasYAML, err := io.ReadAll(reader)
	if err != nil {
		return readResult{}, fmt.Errorf("read canvas yaml: %w", err)
	}

	result := readResult{
		Action:            readActionName,
		CanvasID:          session.CanvasID,
		Source:            "staging",
		VersionID:         liveVersion.ID.String(),
		Summary:           a.summarize(session.OrganizationID, canvas, liveVersion, string(canvasYAML)),
		CanvasYAMLBytes:   len(canvasYAML),
		CanvasYAMLOmitted: !input.IncludeCanvasYAML,
	}

	if input.IncludeCanvasYAML {
		result.CanvasYAML = string(canvasYAML)
	}

	if input.IncludeConsole {
		reader, err := fileReader.Read(ctx, files.ConsoleYAMLPath)
		if err != nil {
			return readResult{}, fmt.Errorf("read console yaml: %w", err)
		}
		consoleYAML, err := io.ReadAll(reader)
		if err != nil {
			return readResult{}, fmt.Errorf("read console yaml: %w", err)
		}
		result.ConsoleYAML = string(consoleYAML)
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
