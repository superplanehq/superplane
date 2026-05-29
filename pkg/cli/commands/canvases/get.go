package canvases

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/canvasresolve"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct {
	draft *bool
}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := canvasresolve.FindCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return err
	}
	if response.Canvas == nil {
		return fmt.Errorf("canvas %q not found", canvasID)
	}

	canvas := *response.Canvas
	if c.draft != nil && *c.draft {
		me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
		if err != nil {
			return err
		}
		currentUserID := strings.TrimSpace(me.User.GetId())
		if currentUserID == "" {
			return fmt.Errorf("current user id not found")
		}

		versionID, err := canvasresolve.FindOwnedDraftVersionID(ctx, canvasID, currentUserID)
		if err != nil {
			return err
		}
		if versionID == "" {
			return fmt.Errorf("draft version not found for current user")
		}

		version, err := canvasresolve.DescribeCanvasVersionByID(ctx, canvasID, versionID)
		if err != nil {
			return err
		}
		if version.Spec != nil {
			canvas.SetSpec(*version.Spec)
		}
	}

	resource := models.CanvasResourceFromCanvas(canvas)
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "ID: %s\n", resource.Metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", resource.Metadata.GetName())
		if url := BuildCanvasURL(ctx, canvas.Metadata.GetOrganizationId(), canvas.Metadata.GetId()); url != "" {
			_, _ = fmt.Fprintf(stdout, "Canvas URL: %s\n", url)
		}
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", len(resource.Spec.GetNodes()))
		_, err := fmt.Fprintf(stdout, "Edges: %d\n", len(resource.Spec.GetEdges()))
		return err
	})
}
