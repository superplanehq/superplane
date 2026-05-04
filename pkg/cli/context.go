package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ContextCommand struct{}

func (c *ContextCommand) Execute(ctx core.CommandContext) error {
	contexts := GetContexts()
	if len(contexts) == 0 {
		return fmt.Errorf("no contexts configured; run superplane connect [BASE_URL] [API_TOKEN]")
	}

	if len(ctx.Args) == 0 {
		if !ctx.Renderer.IsText() || !ctx.IsInteractive() {
			return renderContextsOutput(ctx, contexts)
		}

		selected, err := promptContextSelection(ctx, contexts)
		if err != nil {
			return err
		}

		org := selected.OrganizationID
		if org == "" {
			org = selected.Organization
		}
		switched, err := SwitchContext(selected.URL, org)
		if err != nil {
			return err
		}
		return renderContextOutput(ctx, *switched)
	}

	urlFlag, err := ctx.Cmd.Flags().GetString("url")
	if err != nil {
		return err
	}

	selected, err := SwitchContextByOrganization(ctx.Args[0], urlFlag)
	if err != nil {
		return err
	}
	return renderContextOutput(ctx, *selected)
}

var contextCmd = &cobra.Command{
	Use:   "context [ORG_OR_ID]",
	Short: "Switch CLI context by organization id or name",
	Long: "With ORG_OR_ID, switches to the saved context for that organization (matches id first, then name). " +
		"If the same id or name exists on multiple installations, pass --url with the installation base URL. " +
		"Without arguments, lists contexts and prompts for a selection (same as \"superplane contexts\"). " +
		"To switch using an explicit base URL and organization, use \"superplane contexts BASE_URL ORGANIZATION\".",
	Args: cobra.MaximumNArgs(1),
}

func init() {
	contextCmd.Flags().String("url", "", "installation base URL when multiple contexts match")
	core.Bind(contextCmd, &ContextCommand{}, defaultBindOptions())
	RootCmd.AddCommand(contextCmd)
}
