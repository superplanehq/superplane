package apps

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type listCommand struct{}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.CanvasAPI.CanvasesListCanvases(ctx.Context).Execute()
	if err != nil {
		return err
	}

	canvases := response.GetCanvases()
	if !ctx.Renderer.IsText() {
		apps := make([]map[string]any, len(canvases))
		for i, canvas := range canvases {
			apps[i] = map[string]any{
				"id":          canvas.GetId(),
				"name":        canvas.GetName(),
				"description": canvas.GetDescription(),
				"createdBy":   canvas.GetCreatedBy().Name,
				"createdAt":   canvas.GetCreatedAt().Format(time.RFC3339),
				"updatedAt":   canvas.GetUpdatedAt().Format(time.RFC3339),
				"folderId":    canvas.GetFolderId(),
				"nodes":       len(canvas.GetNodes()),
				"edges":       len(canvas.GetEdges()),
			}
		}
		return ctx.Renderer.Render(apps)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(canvases) == 0 {
			_, err := fmt.Fprintln(stdout, "No apps found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCREATED_AT\tUPDATED_AT")

		for _, canvas := range canvases {
			createdAt := ""
			if canvas.HasCreatedAt() {
				createdAt = canvas.GetCreatedAt().Format(time.RFC3339)
			}
			updatedAt := ""
			if canvas.HasUpdatedAt() {
				updatedAt = canvas.GetUpdatedAt().Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", canvas.GetId(), canvas.GetName(), createdAt, updatedAt)
		}

		return writer.Flush()
	})
}
