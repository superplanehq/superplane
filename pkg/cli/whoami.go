package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type whoamiCommand struct{}

func (w *whoamiCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, _ = fmt.Fprintf(stdout, "ID: %s\n", response.GetId())
			_, _ = fmt.Fprintf(stdout, "Email: %s\n", response.GetEmail())
			_, _ = fmt.Fprintf(stdout, "Organization: %s\n", response.GetOrganizationId())
			return nil
		})
	}

	return ctx.Renderer.Render(response)
}

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "Get information about the currently authenticated user",
	Aliases: []string{"events"},
	Args:    cobra.NoArgs,
}

func init() {
	core.Bind(whoamiCmd, &whoamiCommand{}, defaultBindOptions())
	RootCmd.AddCommand(whoamiCmd)
}
