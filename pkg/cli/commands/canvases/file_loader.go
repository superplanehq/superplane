package canvases

import (
	"fmt"
	"os"
	"strings"

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

func loadCanvasFromFile(ctx core.CommandContext, filePath string) (string, openapi_client.CanvasesCanvas, error) {
	resource, err := parseCanvasResourceFromFile(filePath, "update")
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	fileCanvasID := ""
	if resource.Metadata != nil && resource.Metadata.Id != nil {
		fileCanvasID = strings.TrimSpace(resource.Metadata.GetId())
	}

	activeCanvasID := ""
	if ctx.Config != nil {
		activeCanvasID = strings.TrimSpace(ctx.Config.GetActiveCanvas())
	}
	if activeCanvasID != "" {
		resolved, resolveErr := findCanvasID(ctx, ctx.API, activeCanvasID)
		if resolveErr != nil {
			return "", openapi_client.CanvasesCanvas{}, resolveErr
		}
		activeCanvasID = resolved
	}

	if fileCanvasID != "" && activeCanvasID != "" && fileCanvasID != activeCanvasID {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf(
			"canvas metadata.id %q does not match the active canvas %q; clear the active canvas or fix metadata.id",
			fileCanvasID, activeCanvasID)
	}

	canvasID := fileCanvasID
	if canvasID == "" {
		canvasID = activeCanvasID
	}
	if canvasID == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf(
			"canvas metadata.id is required in the file when no active canvas is set; set one with `superplane canvases active` or add metadata.id to the YAML")
	}

	canvas := models.CanvasFromCanvas(*resource)
	meta := canvas.GetMetadata()
	meta.SetId(canvasID)
	canvas.SetMetadata(meta)

	return canvasID, canvas, nil
}
