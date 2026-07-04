package canvas

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas/models"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
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
	canvasName := strings.TrimSpace(described.Metadata.GetName())
	organizationID := strings.TrimSpace(described.Metadata.GetOrganizationId())

	yamlBytes, err := common.FetchRepositoryFile(ctx, canvasID, common.CanvasYAMLRepositoryPath, "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(yamlBytes)) == "" {
		return fmt.Errorf("app %q has no canvas", canvasID)
	}

	resource, err := models.ParseCanvas(yamlBytes)
	if err != nil {
		return fmt.Errorf("invalid canvas yaml from server: %w", err)
	}
	if resource.Metadata == nil {
		return fmt.Errorf("canvas metadata is required")
	}
	if resource.Metadata.Id == nil || strings.TrimSpace(resource.Metadata.GetId()) == "" {
		resource.Metadata.SetId(canvasID)
	}
	if resource.Metadata.Name == nil || strings.TrimSpace(resource.Metadata.GetName()) == "" {
		if canvasName == "" {
			return fmt.Errorf("canvas metadata.name is required")
		}
		resource.Metadata.SetName(canvasName)
	}
	if organizationID != "" && (resource.Metadata.OrganizationId == nil || strings.TrimSpace(resource.Metadata.GetOrganizationId()) == "") {
		resource.Metadata.SetOrganizationId(organizationID)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "ID: %s\n", resource.Metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", resource.Metadata.GetName())
		if url := common.BuildAppURL(ctx, organizationID, canvasID); url != "" {
			_, _ = fmt.Fprintf(stdout, "App URL: %s\n", url)
		}
		nodeCount := 0
		edgeCount := 0
		if resource.Spec != nil {
			nodeCount = len(resource.Spec.GetNodes())
			edgeCount = len(resource.Spec.GetEdges())
		}
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", nodeCount)
		_, err := fmt.Fprintf(stdout, "Edges: %d\n", edgeCount)
		return err
	})
}
