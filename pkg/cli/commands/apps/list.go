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
	response, _, err := ctx.API.AppAPI.AppsListApps(ctx.Context).Execute()
	if err != nil {
		return err
	}

	apps := response.GetApps()
	if !ctx.Renderer.IsText() {
		summary := make([]map[string]string, len(apps))
		for i, app := range apps {
			metadata := app.GetMetadata()
			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}
			summary[i] = map[string]string{
				"id":          metadata.GetId(),
				"displayName": metadata.GetDisplayName(),
				"slug":        metadata.GetSlug(),
				"createdAt":   createdAt,
			}
		}
		return ctx.Renderer.Render(summary)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(apps) == 0 {
			_, err := fmt.Fprintln(stdout, "No apps found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tDISPLAY_NAME\tSLUG\tCREATED_AT")

		for _, app := range apps {
			metadata := app.GetMetadata()
			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetDisplayName(),
				metadata.GetSlug(),
				createdAt,
			)
		}

		return writer.Flush()
	})
}
