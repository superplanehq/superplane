package console

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type getCommand struct {
	canvasID *string
}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	dashboard, err := fetchDashboard(ctx, canvasID)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(dashboard)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderConsoleSummaryText(stdout, canvasID, dashboard)
	})
}

func fetchDashboard(ctx core.CommandContext, canvasID string) (openapi_client.CanvasesCanvasDashboard, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesGetCanvasDashboard(ctx.Context, canvasID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvasDashboard{}, err
	}
	if response.Dashboard == nil {
		return openapi_client.CanvasesCanvasDashboard{}, nil
	}
	return *response.Dashboard, nil
}

func renderConsoleSummaryText(stdout io.Writer, canvasID string, dashboard openapi_client.CanvasesCanvasDashboard) error {
	if _, err := fmt.Fprintf(stdout, "Canvas ID: %s\n", canvasID); err != nil {
		return err
	}
	if dashboard.HasUpdatedAt() {
		if _, err := fmt.Fprintf(stdout, "Updated:   %s\n", dashboard.GetUpdatedAt().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(stdout, "Panels:    %d\n", len(dashboard.GetPanels())); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Layout:    %d\n", len(dashboard.GetLayout())); err != nil {
		return err
	}

	if len(dashboard.GetPanels()) == 0 {
		_, err := fmt.Fprintln(stdout, "\nNo panels defined.")
		return err
	}

	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tTYPE\tTITLE\tPOSITION\tSIZE"); err != nil {
		return err
	}
	layoutByID := make(map[string]openapi_client.CanvasesDashboardLayoutItem, len(dashboard.GetLayout()))
	for _, item := range dashboard.GetLayout() {
		layoutByID[item.GetI()] = item
	}
	for _, panel := range dashboard.GetPanels() {
		title := panelTitle(panel)
		positionStr, sizeStr := layoutPositionAndSize(layoutByID[panel.GetId()])
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", panel.GetId(), panel.GetType(), title, positionStr, sizeStr); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func panelTitle(panel openapi_client.CanvasesDashboardPanel) string {
	content := panel.GetContent()
	if content == nil {
		return "(no title)"
	}
	if title, ok := content["title"].(string); ok && title != "" {
		return title
	}
	if node, ok := content["node"].(string); ok && node != "" {
		return node
	}
	return "(no title)"
}

func layoutPositionAndSize(item openapi_client.CanvasesDashboardLayoutItem) (string, string) {
	if item.GetI() == "" {
		return "-", "-"
	}
	return fmt.Sprintf("%d,%d", item.GetX(), item.GetY()), fmt.Sprintf("%dx%d", item.GetW(), item.GetH())
}

func valueOf(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
