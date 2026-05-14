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
type InternalReader struct {
	// SerializeCanvasFunc serializes a canvas to proto JSON bytes.
	// Set by the caller to break the import cycle with grpc/actions/canvases.
	SerializeCanvasFunc func(canvasID string) ([]byte, error)
}

// ReadCanvasYAML exports the canvas as CLI-compatible YAML.
func (r *InternalReader) ReadCanvasYAML(canvasID string) ([]byte, error) {
	if r.SerializeCanvasFunc != nil {
		// Use the proper proto serialization path
		jsonBytes, err := r.SerializeCanvasFunc(canvasID)
		if err != nil {
			return nil, err
		}

		var cliCanvas models.Canvas
		cliCanvas.APIVersion = "v1"
		cliCanvas.Kind = "Canvas"
		json.Unmarshal(jsonBytes, &cliCanvas)

		// Strip readme from canvas.yaml — it lives in README.md
		if cliCanvas.Spec != nil {
		}

		return yaml.Marshal(cliCanvas)
	}

	// Fallback: basic DB read (less accurate)
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

	canvasJSON := map[string]interface{}{
		"metadata": map[string]interface{}{
			"id": canvas.ID.String(), "organizationId": canvas.OrganizationID.String(),
			"name": canvas.Name, "description": canvas.Description,
		},
		"spec": map[string]interface{}{"nodes": version.Nodes, "edges": version.Edges},
	}

	var cliCanvas models.Canvas
	cliCanvas.APIVersion = "v1"
	cliCanvas.Kind = "Canvas"
	jsonBytes, _ := json.Marshal(canvasJSON)
	json.Unmarshal(jsonBytes, &cliCanvas)

	return yaml.Marshal(cliCanvas)
}

// ReadReadme returns the canvas readme content.
func (r *InternalReader) ReadReadme(canvasID string) (string, error) {
	// Readme is no longer stored in DB — it lives in git (docs/README.md)
	return "", nil
}

// LaunchpadData holds the exported launchpad state.
type LaunchpadData struct {
	Panels []dbmodels.LaunchpadPanel
	Layout []dbmodels.LaunchpadLayoutItem
}

// ReadLaunchpad returns the canvas launchpad panels and layout.
func (r *InternalReader) ReadLaunchpad(canvasID string) (*LaunchpadData, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas ID: %w", err)
	}

	lp, err := dbmodels.FindCanvasLaunchpad(canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to read launchpad: %w", err)
	}

	return &LaunchpadData{
		Panels: lp.Panels.Data(),
		Layout: lp.Layout.Data(),
	}, nil
}
