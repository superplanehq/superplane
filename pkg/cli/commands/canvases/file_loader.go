package canvases

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func parseCanvasResourceFromFile(filePath string, operation string) (*models.Canvas, error) {
	// #nosec
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource file: %w", err)
	}

	_, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return nil, err
	}

	if kind != models.CanvasKind {
		return nil, fmt.Errorf("unsupported resource kind %q for %s", kind, operation)
	}

	resource, err := models.ParseCanvas(data)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func loadCanvasForCreateFromFile(filePath string) (openapi_client.CanvasesCanvas, *openapi_client.CanvasesCanvasAutoLayout, error) {
	resource, err := parseCanvasResourceFromFile(filePath, "create")
	if err != nil {
		return openapi_client.CanvasesCanvas{}, nil, err
	}

	return models.CanvasFromCanvas(*resource), resource.AutoLayout, nil
}

func loadCanvasFromFile(filePath string) (string, openapi_client.CanvasesCanvas, error) {
	resource, err := parseCanvasResourceFromFile(filePath, "update")
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	if resource.Metadata == nil || resource.Metadata.Id == nil || resource.Metadata.GetId() == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata.id is required for update")
	}

	return resource.Metadata.GetId(), models.CanvasFromCanvas(*resource), nil
}
