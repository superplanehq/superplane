package canvases

import (
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

// ParseAndValidateCanvasYAML parses canvas.yaml text and runs the same registry
// validation as the commit path, returning materialized nodes/edges (carrying
// per-node error/warning messages) without persisting anything. Agent tools use
// it to validate staged edits before staging and to summarize staged content.
func ParseAndValidateCanvasYAML(registry *registry.Registry, organizationID, text string) ([]models.Node, []models.Edge, error) {
	pbCanvas, err := canvasFromYAMLText(text)
	if err != nil {
		return nil, nil, err
	}
	return ParseCanvas(registry, organizationID, pbCanvas)
}
