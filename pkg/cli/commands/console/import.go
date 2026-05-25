package console

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type importCommand struct {
	canvasID *string
	file     *string
	yes      *bool
}

func (c *importCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	resource, err := resourceFromInput(valueOf(c.file), ctx.Cmd.InOrStdin())
	if err != nil {
		return err
	}

	if !confirmReplace(ctx, c.yes, len(resource.Spec.Panels), len(resource.Spec.Layout)) {
		_, err := fmt.Fprintln(ctx.Cmd.OutOrStdout(), "Aborted.")
		return err
	}

	body := resourceToUpdateBody(*resource)
	response, _, err := ctx.API.CanvasAPI.
		CanvasesUpdateCanvasDashboard(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.GetDashboard())
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		dashboard := response.GetDashboard()
		_, err := fmt.Fprintf(stdout, "Console imported (%d panels, %d layout items)\n", len(dashboard.GetPanels()), len(dashboard.GetLayout()))
		return err
	})
}

// confirmReplace returns true when the user accepts the replace-all
// semantics of import/clear (or has passed --yes / is non-interactive).
//
// The Console API is replace-all: importing or clearing wipes any panels and
// layout that are not in the request body. We surface a single confirmation
// prompt for interactive sessions to make that destructive behavior obvious.
func confirmReplace(ctx core.CommandContext, yes *bool, newPanelCount int, newLayoutCount int) bool {
	if yes != nil && *yes {
		return true
	}

	if !ctx.IsInteractive() {
		return true
	}

	_, _ = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Importing replaces all existing panels and layout. New panels: %d, layout items: %d.\nProceed? [y/N]: ", newPanelCount, newLayoutCount)

	var answer string
	_, _ = fmt.Fscanln(ctx.Cmd.InOrStdin(), &answer)
	return answer == "y" || answer == "Y" || answer == "yes" || answer == "YES"
}
