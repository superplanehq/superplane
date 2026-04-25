package canvases

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func validateCanvasUpdateDryRun(
	ctx core.CommandContext,
	canvasID string,
	proposedCanvas openapi_client.CanvasesCanvas,
	targetVersionID string,
) error {
	current, err := describeCanvasVersionByID(ctx, canvasID, targetVersionID)
	if err != nil {
		return err
	}
	if current.Spec == nil {
		return fmt.Errorf("current draft version has no spec")
	}
	proposedSpec := proposedCanvas.GetSpec()

	changeset, err := buildCanvasChangesetFromSpecs(
		current.Spec.GetNodes(),
		current.Spec.GetEdges(),
		proposedSpec.GetNodes(),
		proposedSpec.GetEdges(),
	)
	if err != nil {
		return err
	}
	if len(changeset.GetChanges()) == 0 {
		if !ctx.Renderer.IsText() {
			return ctx.Renderer.Render(map[string]any{
				"valid":   true,
				"message": "no changes compared to the current draft",
			})
		}
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintln(stdout, "Dry-run: no changes compared to the current draft.")
			return err
		})
	}

	body := openapi_client.NewCanvasesValidateCanvasVersionChangesetBody()
	body.SetChangeset(*changeset)

	resp, _, err := ctx.API.CanvasVersionAPI.
		CanvasesValidateCanvasVersionChangeset(ctx.Context, canvasID, targetVersionID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	version := resp.GetVersion()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"valid":   true,
			"version": version,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if err := writeCanvasUpdateSuccessSummary(stdout, version); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout, "Dry-run: validation succeeded (nothing was saved).")
		return err
	})
}
