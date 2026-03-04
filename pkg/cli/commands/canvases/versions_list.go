package canvases

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type versionsListCommand struct{}

func (c *versionsListCommand) Execute(ctx core.CommandContext) error {
	canvasRef := ""
	if len(ctx.Args) == 1 {
		canvasRef = ctx.Args[0]
	}

	canvasID, err := resolveCanvasIDFromArgOrActive(ctx, canvasRef)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesListCanvasVersions(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return err
	}

	versions := response.GetVersions()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(versions)
	}

	liveVersionID, _ := findLiveCanvasVersionID(ctx, canvasID)

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tREVISION\tSTATE\tOWNER\tUPDATED_AT")

		for _, version := range versions {
			metadata := version.GetMetadata()
			state := versionStateLabel(version, liveVersionID)
			owner := versionOwnerLabel(version)
			updatedAt := versionUpdatedAt(version)
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%d\t%s\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetRevision(),
				state,
				owner,
				updatedAt,
			)
		}

		return writer.Flush()
	})
}

func versionStateLabel(version openapi_client.CanvasesCanvasVersion, liveVersionID string) string {
	metadata := version.GetMetadata()
	if metadata.GetIsPublished() {
		if metadata.GetId() == liveVersionID {
			return "live"
		}
		return "live-history"
	}

	return "edit-mode"
}

func versionOwnerLabel(version openapi_client.CanvasesCanvasVersion) string {
	metadata := version.GetMetadata()
	if owner, ok := metadata.GetOwnerOk(); ok && owner != nil {
		name := owner.GetName()
		if name != "" {
			return name
		}
		return owner.GetId()
	}

	if metadata.GetIsPublished() {
		return "system"
	}

	return "-"
}

func versionUpdatedAt(version openapi_client.CanvasesCanvasVersion) string {
	metadata := version.GetMetadata()
	if metadata.HasUpdatedAt() {
		return metadata.GetUpdatedAt().Format(time.RFC3339)
	}
	if metadata.HasPublishedAt() {
		return metadata.GetPublishedAt().Format(time.RFC3339)
	}
	if metadata.HasCreatedAt() {
		return metadata.GetCreatedAt().Format(time.RFC3339)
	}

	return ""
}
