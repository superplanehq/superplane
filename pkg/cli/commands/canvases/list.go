package canvases

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type listCommand struct{}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.CanvasAPI.CanvasesListCanvases(ctx.Context).Execute()
	if err != nil {
		return err
	}

	canvases := response.GetCanvases()
	resources := make([]models.Canvas, 0, len(canvases))
	for _, canvas := range canvases {
		resources = append(resources, models.CanvasResourceFromCanvas(canvas))
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resources)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCREATED_AT")

		for _, canvas := range canvases {
			metadata := canvas.GetMetadata()
			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", metadata.GetId(), metadata.GetName(), createdAt)
		}

		return writer.Flush()
	})
}
