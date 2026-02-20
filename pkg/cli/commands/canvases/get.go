package canvases

import (
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := findCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return err
	}

	resource := models.CanvasResourceFromCanvas(*response.Canvas)
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "ID: %s\n", resource.Metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", resource.Metadata.GetName())
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", len(resource.Spec.GetNodes()))
		_, err := fmt.Fprintf(stdout, "Edges: %d\n", len(resource.Spec.GetEdges()))
		return err
	})
}

func findCanvasID(ctx core.CommandContext, client *openapi_client.APIClient, nameOrID string) (string, error) {
	if _, err := uuid.Parse(nameOrID); err == nil {
		return nameOrID, nil
	}

	return findCanvasIDByName(ctx, client, nameOrID)
}

func findCanvasIDByName(ctx core.CommandContext, client *openapi_client.APIClient, name string) (string, error) {
	response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.CanvasesCanvas
	for _, canvas := range response.GetCanvases() {
		if canvas.Metadata == nil || canvas.Metadata.Name == nil {
			continue
		}
		if *canvas.Metadata.Name == name {
			matches = append(matches, canvas)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("canvas %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple canvases named %q found", name)
	}

	if matches[0].Metadata == nil || matches[0].Metadata.Id == nil {
		return "", fmt.Errorf("canvas %q is missing an id", name)
	}

	return *matches[0].Metadata.Id, nil
}
