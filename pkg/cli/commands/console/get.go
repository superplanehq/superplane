package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/canvasresolve"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct {
	draft *bool
}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("get accepts at most one positional argument")
	}

	canvasArg := ""
	if len(ctx.Args) == 1 {
		canvasArg = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := canvasresolve.ResolveCanvasNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	canvasName, err := lookupCanvasName(ctx, canvasID)
	if err != nil {
		return err
	}

	useDraft := c.draft != nil && *c.draft
	versionID := ""
	if useDraft {
		versionID, err = resolveCurrentUserDraftVersionID(ctx, canvasID)
		if err != nil {
			return err
		}
	}

	request := ctx.API.CanvasAPI.CanvasesGetCanvasDashboard(ctx.Context, canvasID)
	if versionID != "" {
		request = request.VersionId(versionID)
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}
	if response.Dashboard == nil {
		return fmt.Errorf("canvas %q has no dashboard", canvasID)
	}

	dashboard := *response.Dashboard
	resource := consoleYAMLFromAPI(canvasName, dashboard)

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		source := "live"
		if useDraft {
			source = "draft"
		}
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", canvasName)
		_, _ = fmt.Fprintf(stdout, "Canvas ID: %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Source: %s\n", source)
		if versionID := strings.TrimSpace(dashboard.GetVersionId()); versionID != "" {
			_, _ = fmt.Fprintf(stdout, "Version ID: %s\n", versionID)
		}
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

// resolveCurrentUserDraftVersionID returns the id of the current user's
// existing draft version. Unlike the canvas update flow we do not create
// a draft on read: an absent draft is an error so users get an immediate
// signal that there is nothing to read yet.
func resolveCurrentUserDraftVersionID(ctx core.CommandContext, canvasID string) (string, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}
	currentUserID := strings.TrimSpace(me.User.GetId())
	if currentUserID == "" {
		return "", fmt.Errorf("current user id not found")
	}

	versionID, err := canvasresolve.FindOwnedDraftVersionID(ctx, canvasID, currentUserID)
	if err != nil {
		return "", err
	}
	if versionID == "" {
		return "", fmt.Errorf("draft version not found for current user")
	}
	return versionID, nil
}
