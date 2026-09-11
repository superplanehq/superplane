package workspaces

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type activeCommand struct{}

func (c *activeCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) == 1 {
		return c.setActive(ctx, ctx.Args[0])
	}

	if !ctx.IsInteractive() || !ctx.Renderer.IsText() {
		return c.printActive(ctx)
	}

	return c.setActiveInteractively(ctx)
}

func (c *activeCommand) setActive(ctx core.CommandContext, nameOrID string) error {
	workspaceID, err := FindWorkspaceID(ctx, nameOrID)
	if err != nil {
		return err
	}
	return ctx.Config.SetActiveWorkspace(workspaceID)
}

func (c *activeCommand) printActive(ctx core.CommandContext) error {
	active := strings.TrimSpace(ctx.Config.GetActiveWorkspace())
	if active == "" {
		return fmt.Errorf("no active workspace; pass a name, key, or id, or run interactively")
	}
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]string{"id": active})
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintln(stdout, active)
		return err
	})
}

func (c *activeCommand) setActiveInteractively(ctx core.CommandContext) error {
	response, _, err := ctx.API.FactoryAPI.FactoriesListFactories(ctx.Context).Execute()
	if err != nil {
		return err
	}

	workspaces := response.GetFactories()
	if len(workspaces) == 0 {
		return fmt.Errorf("no workspaces found")
	}

	err = ctx.Renderer.RenderText(func(stdout io.Writer) error {
		for i, ws := range workspaces {
			prefix := " "
			if ws.GetId() == ctx.Config.GetActiveWorkspace() {
				prefix = "*"
			}
			_, _ = fmt.Fprintf(stdout, "%s %d. %s (%s)\n", prefix, i+1, ws.GetName(), ws.GetId())
		}
		_, _ = fmt.Fprint(stdout, "Select a workspace number: ")
		return nil
	})
	if err != nil {
		return err
	}

	reader := bufio.NewReader(ctx.Cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read selected workspace: %w", err)
	}

	selectedIndex, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return fmt.Errorf("invalid workspace selection %q", strings.TrimSpace(input))
	}
	if selectedIndex < 1 || selectedIndex > len(workspaces) {
		return fmt.Errorf("workspace selection must be between 1 and %d", len(workspaces))
	}

	selected := workspaces[selectedIndex-1]
	if !selected.HasId() {
		return fmt.Errorf("selected workspace is missing an id")
	}
	return ctx.Config.SetActiveWorkspace(selected.GetId())
}
