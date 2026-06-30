package actions

import (
	"context"
	"fmt"
	"strings"

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

	versionID := ""
	source := "live"
	var draft *models.CanvasVersion
	if shouldReadDraft(input) {
		draft, err = resolveReadableDraftVersion(canvasID, uuid.MustParse(session.UserID), input)
		if err != nil {
			return readResult{}, fmt.Errorf("load draft: %w", err)
		}
	}
	if draft != nil {
		versionID = draft.ID.String()
		source = "draft"
	}

	// Read the effective staged content (staged edits when present, the
	// materialized version row otherwise) so the agent observes the same draft
	// state the UI edits and the same edits it stages through patch_draft.
	canvasYAML, err := readRepositorySpecFileForSource(ctx, session.OrganizationID, session.CanvasID, versionID, canvasRepository.CanvasYAMLRepositoryPath, source)
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
		Summary:    a.summarize(session.OrganizationID, canvas, version, source, canvasYAML),
		CanvasYAML: canvasYAML,
	}

	if draft != nil {
		result.Draft = &draftResult{
			VersionID:   draft.ID.String(),
			DisplayName: draft.DisplayName,
			BranchName:  draft.GitBranch,
		}
	}

	if input.IncludeConsole {
		consoleYAML, consoleErr := readRepositorySpecFileForSource(ctx, session.OrganizationID, session.CanvasID, versionID, canvasRepository.ConsoleYAMLRepositoryPath, source)
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

func readRepositorySpecFileForSource(ctx context.Context, organizationID, canvasID, versionID, path, source string) (string, error) {
	if source == "draft" {
		return canvasRepository.ReadRepositorySpecFileStaged(ctx, organizationID, canvasID, versionID, path)
	}
	return canvasRepository.ReadRepositorySpecFile(ctx, organizationID, canvasID, versionID, path)
}

// summarize derives the canvas summary from the YAML the read returns. A draft
// read serves effective staged YAML, which staging never materializes into the
// version row, so the summary is parsed from that YAML. A live read keeps the
// materialized version-row summary and falls back to it when staged YAML cannot
// be parsed (for example after the UI stages content the agent never validated).
func (a readAction) summarize(organizationID string, canvas *models.Canvas, version *models.CanvasVersion, source, canvasYAML string) summary {
	if source != "draft" {
		return summarizeCanvasVersion(canvas, version)
	}

	nodes, edges, err := canvasRepository.ParseAndValidateCanvasYAML(a.deps.Registry, organizationID, canvasYAML)
	if err != nil {
		return summarizeCanvasVersion(canvas, version)
	}

	name := ""
	if canvas != nil {
		name = canvas.Name
	}
	if name == "" && version != nil {
		name = version.Name
	}
	return summarizeParsedCanvas(name, nodes, edges)
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

func shouldReadDraft(input Input) bool {
	if input.UseDraft != nil && !*input.UseDraft {
		return false
	}
	if strings.TrimSpace(input.VersionID) != "" || strings.TrimSpace(input.DraftVersionID) != "" {
		return true
	}
	return true
}
