package workspaces

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type listCommand struct{}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.FactoryAPI.FactoriesListFactories(ctx.Context).Execute()
	if err != nil {
		return err
	}

	workspaces := response.GetFactories()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(workspaces)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderWorkspaceListText(stdout, workspaces, activeWorkspaceID(ctx))
	})
}

func renderWorkspaceListText(stdout io.Writer, workspaces []openapi_client.FactoriesFactory, activeID string) error {
	if len(workspaces) == 0 {
		_, err := fmt.Fprintln(stdout, "No workspaces found.")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "KEY\tNAME\tACTIVE\tID")
	for _, workspace := range workspaces {
		active := ""
		if activeID != "" && workspace.GetId() == activeID {
			active = "*"
		}
		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			workspace.GetKey(),
			workspace.GetName(),
			active,
			workspace.GetId(),
		)
	}
	return writer.Flush()
}

func activeWorkspaceID(ctx core.CommandContext) string {
	if ctx.Config == nil {
		return ""
	}
	return ctx.Config.GetActiveWorkspace()
}
