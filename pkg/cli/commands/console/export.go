package console

import (
	"fmt"
	"io"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type exportCommand struct {
	canvasID *string
	file     *string
}

func (c *exportCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	dashboard, err := fetchDashboard(ctx, canvasID)
	if err != nil {
		return err
	}

	canvasName := findCanvasName(ctx, canvasID)
	resource := dashboardToResource(dashboard, canvasName)
	resource.Metadata.CanvasID = canvasID

	yamlBytes, err := renderConsoleResourceYAML(resource)
	if err != nil {
		return err
	}

	target := valueOf(c.file)
	if target == "" || target == "-" {
		_, err := io.WriteString(ctx.Cmd.OutOrStdout(), string(yamlBytes))
		return err
	}

	if err := os.WriteFile(target, yamlBytes, 0o600); err != nil {
		return fmt.Errorf("failed to write console yaml to %s: %w", target, err)
	}

	if ctx.Renderer.IsText() {
		_, err := fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Console exported to %s\n", target)
		return err
	}

	return ctx.Renderer.Render(map[string]string{
		"canvasId": canvasID,
		"file":     target,
	})
}
