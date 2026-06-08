package apps

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type deleteCommand struct{}

func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	nameOrID := ctx.Args[0]

	appID, err := common.FindAppID(ctx, ctx.API, nameOrID)
	if err != nil {
		return err
	}

	_, _, err = ctx.API.CanvasAPI.
		CanvasesDeleteCanvas(ctx.Context, appID).
		Execute()
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

// NewDeleteCommand registers app deletion under `apps delete`.
func NewDeleteCommand(options core.BindOptions) *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete <name-or-id>",
		Short: "Delete an app",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)
	return deleteCmd
}
