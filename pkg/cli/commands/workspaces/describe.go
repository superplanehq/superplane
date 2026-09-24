package workspaces

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func resolveDescribeWorkspace(ctx core.CommandContext, workspaceFlag *string) (string, error) {
	flagValue := ""
	if workspaceFlag != nil {
		flagValue = strings.TrimSpace(*workspaceFlag)
	}

	positional := ""
	if len(ctx.Args) == 1 {
		positional = strings.TrimSpace(ctx.Args[0])
	}

	if flagValue != "" && positional != "" {
		return "", fmt.Errorf("pass a workspace argument or --workspace, not both")
	}

	ref := flagValue
	if ref == "" {
		ref = positional
	}
	return ResolveWorkspaceID(ctx, ref)
}

type describeCommand struct {
	workspace *string
}

func (c *describeCommand) Execute(ctx core.CommandContext) error {
	workspaceRef, err := resolveDescribeWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.FactoryAPI.FactoriesDescribeFactory(ctx.Context, workspaceRef).Execute()
	if err != nil {
		return err
	}

	workspace := response.GetFactory()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(workspace)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderWorkspaceDescribeText(stdout, workspace)
	})
}

func renderWorkspaceDescribeText(stdout io.Writer, workspace openapi_client.FactoriesFactory) error {
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintf(writer, "Key\t%s\n", workspace.GetKey())
	_, _ = fmt.Fprintf(writer, "Name\t%s\n", workspace.GetName())
	_, _ = fmt.Fprintf(writer, "ID\t%s\n", workspace.GetId())
	_, _ = fmt.Fprintf(writer, "Lines\t%s\n", formatWorkspaceLineNames(workspace.GetLines()))
	return writer.Flush()
}

func formatWorkspaceLineNames(lines []openapi_client.FactoriesFactoryLine) string {
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line.GetName())
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}
