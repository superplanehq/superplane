package extensions

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/commands/extensions/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ListExtensionsCommand struct{}

func (c *ListExtensionsCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.ExtensionAPI.ExtensionsListExtensions(ctx.Context).Execute()
	if err != nil {
		return err
	}

	extensions := response.GetExtensions()
	resources := make([]models.Extension, 0, len(extensions))
	for _, extension := range extensions {
		resources = append(resources, models.ExtensionResourceFromExtension(extension))
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resources)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCREATED_AT")

		for _, extension := range extensions {
			metadata := extension.GetMetadata()
			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", metadata.GetId(), metadata.GetName(), createdAt)
		}

		return writer.Flush()
	})
}
