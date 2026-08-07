package factories

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
	factoryID, err := FindFactoryID(ctx, nameOrID)
	if err != nil {
		return err
	}
	return ctx.Config.SetActiveFactory(factoryID)
}

func (c *activeCommand) printActive(ctx core.CommandContext) error {
	active := strings.TrimSpace(ctx.Config.GetActiveFactory())
	if active == "" {
		return fmt.Errorf("no active factory; pass a name or id, or run interactively")
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

	factories := response.GetFactories()
	if len(factories) == 0 {
		return fmt.Errorf("no factories found")
	}

	err = ctx.Renderer.RenderText(func(stdout io.Writer) error {
		for i, factory := range factories {
			prefix := " "
			if factory.GetId() == ctx.Config.GetActiveFactory() {
				prefix = "*"
			}
			_, _ = fmt.Fprintf(stdout, "%s %d. %s (%s)\n", prefix, i+1, factory.GetName(), factory.GetId())
		}
		_, _ = fmt.Fprint(stdout, "Select a factory number: ")
		return nil
	})
	if err != nil {
		return err
	}

	reader := bufio.NewReader(ctx.Cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read selected factory: %w", err)
	}

	selectedIndex, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return fmt.Errorf("invalid factory selection %q", strings.TrimSpace(input))
	}
	if selectedIndex < 1 || selectedIndex > len(factories) {
		return fmt.Errorf("factory selection must be between 1 and %d", len(factories))
	}

	selected := factories[selectedIndex-1]
	if !selected.HasId() {
		return fmt.Errorf("selected factory is missing an id")
	}
	return ctx.Config.SetActiveFactory(selected.GetId())
}
