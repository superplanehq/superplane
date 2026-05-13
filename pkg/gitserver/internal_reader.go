package gitserver

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/database"
	dbmodels "github.com/superplanehq/superplane/pkg/models"
	"gopkg.in/yaml.v3"
)

// InternalReader reads canvas data directly from the database,
// no API token needed. Used by reverse sync and bootstrap.
type InternalReader struct{}

// ReadCanvasYAML exports the canvas as CLI-compatible YAML.
func (r *InternalReader) ReadCanvasYAML(canvasID string) ([]byte, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas ID: %w", err)
	}

	canvas, err := dbmodels.FindCanvasWithoutOrgScope(canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	version, err := dbmodels.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
	if err != nil {
		return nil, fmt.Errorf("no live version: %w", err)
	}

	// Build CLI-compatible YAML structure
	cliCanvas := models.Canvas{
		APIVersion: "v1",
		Kind:       "Canvas",
	}

	// Marshal canvas + version data through JSON to populate the openapi model
	canvasJSON := map[string]interface{}{
		"metadata": map[string]interface{}{
			"id":             canvas.ID.String(),
			"organizationId": canvas.OrganizationID.String(),
			"name":           canvas.Name,
			"description":    canvas.Description,
		},
		"spec": map[string]interface{}{
			"nodes": version.Nodes,
			"edges": version.Edges,
		},
	}

	jsonBytes, _ := json.Marshal(canvasJSON)
	json.Unmarshal(jsonBytes, &cliCanvas)

	return yaml.Marshal(cliCanvas)
}

// ReadReadme returns the canvas readme content.
func (r *InternalReader) ReadReadme(canvasID string) (string, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return "", fmt.Errorf("invalid canvas ID: %w", err)
	}

	canvas, err := dbmodels.FindCanvasWithoutOrgScope(canvasUUID)
	if err != nil {
		return "", fmt.Errorf("canvas not found: %w", err)
	}

	version, err := dbmodels.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
	if err != nil {
		return "", fmt.Errorf("no live version: %w", err)
	}

	return version.Readme, nil
}
