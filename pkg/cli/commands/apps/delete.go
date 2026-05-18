package apps

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type deleteCommand struct {
	yes *bool
}

func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	nameOrID := ctx.Args[0]

	appID, err := findAppID(ctx, nameOrID)
	if err != nil {
		return err
	}

	confirmed, err := c.confirmDeletion(ctx, nameOrID)
	if err != nil {
		return err
	}
	if !confirmed {
		_, _ = fmt.Fprintln(ctx.Cmd.OutOrStdout(), "Deletion cancelled.")
		return nil
	}

	_, _, err = ctx.API.AppAPI.AppsDeleteApp(ctx.Context, appID).Execute()
	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "App deleted: %s\n", nameOrID)
			return err
		})
	}

	return ctx.Renderer.Render(map[string]string{
		"id":      appID,
		"deleted": "true",
	})
}

func (c *deleteCommand) confirmDeletion(ctx core.CommandContext, nameOrID string) (bool, error) {
	if c.yes != nil && *c.yes {
		return true, nil
	}

	if !ctx.IsInteractive() {
		// Non-interactive with no --yes flag: require explicit confirmation.
		return false, fmt.Errorf("deletion requires confirmation; pass --yes to confirm non-interactively")
	}

	err := ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Are you sure you want to delete app %q? This cannot be undone. [y/N]: ", nameOrID)
		return err
	})
	if err != nil {
		return false, err
	}

	reader := bufio.NewReader(ctx.Cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	answer := strings.TrimSpace(strings.ToLower(input))
	return answer == "y" || answer == "yes", nil
}
