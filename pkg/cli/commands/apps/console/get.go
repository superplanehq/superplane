package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("get accepts at most one positional argument")
	}

	canvasArg := ""
	if len(ctx.Args) == 1 {
		canvasArg = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	canvasName, err := lookupCanvasName(ctx, canvasID)
	if err != nil {
		return err
	}

	yamlBytes, err := common.FetchRepositoryFile(ctx, canvasID, common.ConsoleYAMLRepositoryPath, "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(yamlBytes)) == "" {
		return fmt.Errorf("app %q has no console", canvasID)
	}

	resource, err := ParseConsoleYAML(yamlBytes)
	if err != nil {
		return fmt.Errorf("invalid console yaml from server: %w", err)
	}
	if strings.TrimSpace(resource.Metadata.Name) == "" {
		resource.Metadata.Name = canvasName
	}
	if strings.TrimSpace(resource.Metadata.CanvasID) == "" {
		resource.Metadata.CanvasID = canvasID
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "App: %s\n", canvasName)
		_, _ = fmt.Fprintf(stdout, "App ID: %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", len(resource.Spec.Panels))
		_, err := fmt.Fprintf(stdout, "Layout items: %d\n", len(resource.Spec.Layout))
		return err
	})
}

// lookupCanvasName fetches the canvas to populate `metadata.name` on the
// exported YAML. The name is informational on import, so a fetch failure
// here returns a clear error rather than silently falling back.
func lookupCanvasName(ctx core.CommandContext, canvasID string) (string, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return "", err
	}
	if response.Canvas == nil || response.Canvas.Metadata == nil {
		return "", fmt.Errorf("canvas %q not found", canvasID)
	}
	return response.Canvas.Metadata.GetName(), nil
}
