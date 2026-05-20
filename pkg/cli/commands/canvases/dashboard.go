package canvases

import (
	"fmt"
	"io"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type dashboardGetCommand struct{}

type dashboardUpdateCommand struct {
	file *string
}

func (c *dashboardGetCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := findCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	canvas, err := describeCanvas(ctx, canvasID)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesGetCanvasDashboard(ctx.Context, canvasID).Execute()
	if err != nil {
		return err
	}
	if response == nil || response.Dashboard == nil {
		return fmt.Errorf("dashboard for canvas %q not found", canvasID)
	}

	canvasName := ""
	if canvas.Metadata != nil {
		canvasName = canvas.Metadata.GetName()
	}

	resource := models.DashboardResourceFromDashboard(*response.Dashboard, canvasName)
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Canvas ID: %s\n", resource.Metadata.CanvasID)
		if resource.Metadata.Name != "" {
			_, _ = fmt.Fprintf(stdout, "Canvas Name: %s\n", resource.Metadata.Name)
		}
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(resource.Spec.Panels))
		_, err := fmt.Fprintf(stdout, "Layout Items: %d\n", len(resource.Spec.Layout))
		return err
	})
}

func (c *dashboardUpdateCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := findCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	resource, err := parseDashboardResourceFromFile(filePath)
	if err != nil {
		return err
	}

	body := models.UpdateDashboardRequestFromDashboard(*resource)
	response, _, err := ctx.API.CanvasAPI.
		CanvasesUpdateCanvasDashboard(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}
	if response == nil || response.Dashboard == nil {
		return fmt.Errorf("failed to update dashboard: the server returned an empty response")
	}

	rendered := models.DashboardResourceFromDashboard(*response.Dashboard, resource.Metadata.Name)
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(rendered)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Dashboard updated for canvas %s\n", rendered.Metadata.CanvasID)
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(rendered.Spec.Panels))
		_, err := fmt.Fprintf(stdout, "Layout Items: %d\n", len(rendered.Spec.Layout))
		return err
	})
}

func parseDashboardResourceFromFile(filePath string) (*models.Dashboard, error) {
	if filePath == "" {
		return nil, fmt.Errorf("dashboard file is required")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dashboard file: %w", err)
	}

	_, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return nil, err
	}
	if kind != models.DashboardKind {
		return nil, fmt.Errorf("unsupported resource kind %q for dashboard update", kind)
	}

	return models.ParseDashboard(data)
}

func describeCanvas(ctx core.CommandContext, canvasID string) (openapi_client.CanvasesCanvas, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvas{}, err
	}
	if response == nil || response.Canvas == nil {
		return openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas %q not found", canvasID)
	}

	return *response.Canvas, nil
}
