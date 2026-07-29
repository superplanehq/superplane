package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/yaml"
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

	// Read path: pre-cap consoles are grandfathered so this parse must
	// not fail when a stored console exceeds newer limits. The strict
	// validator only runs on the save/import paths (`apps console set`,
	// commit, install).
	console, err := yaml.ConsoleFromYMLLenient(yamlBytes)
	if err != nil {
		return fmt.Errorf("invalid console yaml from server: %w", err)
	}
	if strings.TrimSpace(console.Metadata.Name) == "" {
		console.Metadata.Name = canvasName
	}
	if strings.TrimSpace(console.Metadata.CanvasID) == "" {
		console.Metadata.CanvasID = canvasID
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(console)
	}

	pages := console.Pages()
	var panelCount, layoutCount int
	for _, page := range pages {
		panelCount += len(page.Panels)
		layoutCount += len(page.Layout)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "App: %s\n", canvasName)
		_, _ = fmt.Fprintf(stdout, "App ID: %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Pages: %d\n", len(pages))
		_, _ = fmt.Fprintf(stdout, "Panels: %d\n", panelCount)
		_, err := fmt.Fprintf(stdout, "Layout items: %d\n", layoutCount)
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
