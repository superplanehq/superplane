package canvas

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

	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return err
	}
	if response.Canvas == nil || response.Canvas.Metadata == nil {
		return fmt.Errorf("canvas %q not found", canvasID)
	}

	described := response.Canvas
	organizationID := strings.TrimSpace(described.Metadata.GetOrganizationId())

	yamlBytes, err := common.FetchRepositoryFile(ctx, canvasID, common.CanvasYAMLRepositoryPath, "")
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(yamlBytes)) == "" {
		return fmt.Errorf("app %q has no canvas", canvasID)
	}

	canvas, err := yaml.CanvasFromYAML(yamlBytes)
	if err != nil {
		return fmt.Errorf("invalid canvas yaml: %w", err)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(canvas)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "ID: %s\n", canvas.Metadata.ID)
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", canvas.Metadata.Name)
		if url := common.BuildAppURL(ctx, organizationID, canvasID); url != "" {
			_, _ = fmt.Fprintf(stdout, "App URL: %s\n", url)
		}
		nodeCount := 0
		edgeCount := 0
		if canvas.Spec != nil {
			nodeCount = len(canvas.Spec.Nodes)
			edgeCount = len(canvas.Spec.Edges)
		}
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", nodeCount)
		_, err := fmt.Fprintf(stdout, "Edges: %d\n", edgeCount)
		return err
	})
}
