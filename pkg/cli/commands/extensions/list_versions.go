package extensions

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/commands/extensions/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ListVersionsCommand struct {
	ExtensionID string
}

func (c *ListVersionsCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.ExtensionAPI.ExtensionsListVersions(ctx.Context, c.ExtensionID).Execute()
	if err != nil {
		return err
	}

	versions := response.GetVersions()
	resources := make([]models.ExtensionVersion, 0, len(versions))
	for _, version := range versions {
		resources = append(resources, models.ExtensionVersion{
			APIVersion: "v1",
			Kind:       models.ExtensionVersionKind,
			Metadata:   version.Metadata,
			Status:     version.Status,
		})
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resources)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tVERSION\tSTATE\tCREATED_AT")

		for _, v := range versions {
			metadata := v.GetMetadata()
			status := v.GetStatus()

			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}

			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetVersion(),
				string(status.GetState()),
				createdAt,
			)
		}

		return writer.Flush()
	})
}
